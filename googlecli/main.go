package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"

	"github.com/vorteil/direktiv-apps/pkg/direktivapps"
)

// InputContainerDetails ...
type InputContainerDetails struct {
	ServiceAccountKey string   `json:"serviceAccountKey"`
	Command           []string `json:"command"`
	Project           string   `json:"project"`
}

func main() {

	g := direktivapps.ActionError{
		ErrorCode:    "com.googlecli.error",
		ErrorMessage: "",
	}

	obj := new(InputContainerDetails)
	direktivapps.ReadIn(obj, g)

	// json format flag
	obj.Command = append(obj.Command, `--format="json"`)

	if obj.Project == "" {
		g.ErrorMessage = "input project cannot be empty"
		direktivapps.WriteError(g)
	}

	err := ioutil.WriteFile("/key.json", []byte(obj.ServiceAccountKey), 0644)
	if err != nil {
		g.ErrorMessage = fmt.Sprintf("could not write key: %s", err)
		direktivapps.WriteError(g)
	}

	cmd := exec.Command("/root/google-cloud-sdk/bin/gcloud", "auth", "activate-service-account", "--key-file", "/key.json")
	resp, err := cmd.CombinedOutput()
	if err != nil {
		if len(resp) > 0 {
			g.ErrorMessage = fmt.Sprintf("failed auth: %s", resp)
		} else {
			g.ErrorMessage = fmt.Sprintf("failed auth: %s", err.Error())
		}
		direktivapps.WriteError(g)
	}

	cmd = exec.Command("/root/google-cloud-sdk/bin/gcloud", "config", "set", "project", obj.Project)
	resp, err = cmd.CombinedOutput()
	if err != nil {
		if len(resp) > 0 {
			g.ErrorMessage = fmt.Sprintf("invalid project: %s", resp)
		} else {
			g.ErrorMessage = fmt.Sprintf("invalid project: %s", err.Error())
		}
		direktivapps.WriteError(g)
	}

	cmd = exec.Command("/root/google-cloud-sdk/bin/gcloud", obj.Command...)
	resp, err = cmd.CombinedOutput()
	if err != nil {
		if len(resp) > 0 {
			g.ErrorMessage = fmt.Sprintf("%s", resp)
		} else {
			g.ErrorMessage = fmt.Sprintf("%s", err.Error())
		}
		direktivapps.WriteError(g)
	}

	direktivapps.WriteOut(resp, g)

}
