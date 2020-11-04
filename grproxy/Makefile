GOPATH := ${PWD}:${GOPATH}
export GOPATH

default: deps build

deps:
	go get github.com/samuel/go-zookeeper/zk

build:
	go build -tags netgo -v -o ./bin/grproxy ./src/grproxy

docker-build:
	docker build -t grproxy-tud .

docker-run:
	docker run -p 127.0.0.1:80:8080  -it --network backend  grproxy-tud:latest