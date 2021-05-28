TARGETNAME=proxy
GOOS=linux
GOARCH=amd64
IMG ?= localhost:32000/proxy:latest

all: format test build clean

test:
	go test -v . 

format:
	gofmt -w .

build:
	mkdir -p releases
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 CGO_LDFLAGS="-static" go build -mod=vendor -ldflags "-s -w" -v -o releases/$(TARGETNAME) .

clean:
	go clean -i

docker-build:
	docker build -f $(shell pwd)/docker/Dockerfile -t ${IMG} .

docker-push:
	docker push ${IMG}
