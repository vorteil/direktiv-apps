REPOSITORY := vorteil

# dependencies
.PHONY: dependencies
dependencies:
	echo "fetching dependencies"
	mkdir -p ./docker/cli-plugins
	wget https://github.com/christian-korneck/docker-pushrm/releases/download/v1.7.0/docker-pushrm_linux_amd64
	cp ./docker-pushrm_linux_amd64 ~/.docker/cli-plugins/docker-pushrm
	chmod +x ~/.docker/cli-plugins/docker-pushrm

# build a singular container using provided environment variable
.PHONY: build-singular
build-singular:
	echo "building ${CONTAINER}"	
	docker build ${CONTAINER} --tag ${REPOSITORY}/${CONTAINER}:latest
	docker build ${CONTAINER} --tag ${REPOSITORY}/${CONTAINER}:${VERSION}
	docker push ${REPOSITORY}/${CONTAINER}:latest
	docker push ${REPOSITORY}/${CONTAINER}:${VERSION}
	cd ${CONTAINER} && docker pushrm docker.io/${REPOSITORY}/${CONTAINER}

# build all containers using provided version variable
