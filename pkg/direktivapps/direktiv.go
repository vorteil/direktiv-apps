package direktivapps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

const (
	DirektivActionIDHeader    = "Direktiv-ActionID"
	DirektivInstanceIDHeader  = "Direktiv-InstanceID"
	DirektivExchangeKeyHeader = "Direktiv-ExchangeKey"
	DirektivPingAddrHeader    = "Direktiv-PingAddr"
	DirektivTimeoutHeader     = "Direktiv-Timeout"
	DirektivStepHeader        = "Direktiv-Step"
	DirektivResponseHeader    = "Direktiv-Response"

	DirektivErrorCodeHeader    = "Direktiv-ErrorCode"
	DirektivErrorMessageHeader = "Direktiv-ErrorMessage"
)

// ActionError is a struct Direktiv uses to report application errors.
type ActionError struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

const outPath = "/direktiv-data/data.out"
const dataInPath = "/direktiv-data/data.in"
const errorPath = "/direktiv-data/error.json"

// Respond with error
func RespondWithError(w http.ResponseWriter, code string, err string) {
	w.Header().Set(DirektivErrorCodeHeader, code)
	w.Header().Set(DirektivErrorMessageHeader, err)
}

// Respond writes out to the responsewriter the json marshalled data
func Respond(w http.ResponseWriter, data []byte) {
	w.Write(data)
}

// Unmarshal reads the req body and unmarshals the data
func Unmarshal(obj interface{}, r *http.Request) (string, error) {

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	rdr := bytes.NewReader(data)
	dec := json.NewDecoder(rdr)

	dec.DisallowUnknownFields()

	err = dec.Decode(obj)
	if err != nil {
		return "", err
	}

	return r.Header.Get(DirektivActionIDHeader), nil
}

// StartServer starts a new server
func StartServer(f func(w http.ResponseWriter, r *http.Request)) {

	fmt.Println("Starting server")

	r := mux.NewRouter()
	r.HandleFunc("/", cancelHandler).Methods(http.MethodDelete)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		aid := r.Header.Get(DirektivActionIDHeader)
		if aid == "" {
			// cant handle a DELETE request with no specific AID
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		r = r.WithContext(ctx)

		reqMap.Store(aid, cancel)
		f(w, r)
		reqMap.Delete(aid)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	go func() {
		<-sigs
		ShutDown(srv)
	}()

	srv.ListenAndServe()
}

var reqMap *RequestMap

func init() {
	reqMap = new(RequestMap)
	reqMap.internal = make(map[string]context.CancelFunc)
}

func cancelHandler(w http.ResponseWriter, r *http.Request) {

	aid := r.Header.Get(DirektivActionIDHeader)
	if aid == "" {
		// cant handle a DELETE request with no specific AID
		return
	}

	reqMap.Delete(aid)
}

// ShutDown turns off the server
func ShutDown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// Log sends a string to log via kubernetes
func Log(aid, l string) {
	if aid == "Development" || aid == "development" {
		fmt.Println(l)
	} else {
		http.Post(fmt.Sprintf("http://localhost:8889/log?aid=%s", aid), "plain/text", strings.NewReader(l))
	}
}

// LogDouble logs to direktiv and stdout
func LogDouble(aid, l string) {
	if aid == "Development" || aid == "development" {
		fmt.Println(l)
	} else {
		fmt.Println(l)
		http.Post(fmt.Sprintf("http://localhost:8889/log?aid=%s", aid), "plain/text", strings.NewReader(l))
	}
}

// ReadIn reads data from dataInPath and returns struct provided with json fields
func ReadIn(obj interface{}, g ActionError) {
	f, err := os.Open(dataInPath)
	if err != nil {
		g.ErrorMessage = err.Error()
		WriteError(g)
	}

	defer f.Close()

	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()

	err = dec.Decode(obj)
	if err != nil {
		g.ErrorMessage = err.Error()
		WriteError(g)
	}
}

// WriteError writes an error to errorPath
func WriteError(g ActionError) {
	b, _ := json.Marshal(g)
	ioutil.WriteFile(errorPath, b, 0755)
	os.Exit(0)
}

// WriteOut writes out data to outPath
func WriteOut(by []byte, g ActionError) {
	var err error
	err = ioutil.WriteFile(outPath, by, 0755)
	if err != nil {
		g.ErrorMessage = err.Error()
		WriteError(g)
	}
	os.Exit(0)
}

// Sync map
type RequestMap struct {
	sync.RWMutex
	internal map[string]context.CancelFunc
}

// Load ..
func (rm *RequestMap) Load(key string) (value context.CancelFunc, ok bool) {
	rm.RLock()
	res, ok := rm.internal[key]
	rm.RUnlock()
	return res, ok
}

// Delete ..
func (rm *RequestMap) Delete(key string) {

	cancel, ok := rm.Load(key)
	if !ok {
		return
	}

	rm.Lock()
	cancel()
	delete(rm.internal, key)
	rm.Unlock()
}

// Store ..
func (rm *RequestMap) Store(key string, value context.CancelFunc) {
	rm.Lock()
	rm.internal[key] = value
	rm.Unlock()
}
