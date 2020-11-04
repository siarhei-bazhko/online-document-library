GOPATH := ${PWD}:${GOPATH}
export GOPATH

default: deps build

deps:
	go get github.com/gorilla/mux
	go get github.com/samuel/go-zookeeper/zk
build:
	go build -tags netgo -v -o ./bin/gserve ./src/gserve/

docker-build:
	docker build -t gserve-tud .

docker-run:
	docker run -p 127.0.0.1:8888:8888  -it --network backend  gserve-tud:latest