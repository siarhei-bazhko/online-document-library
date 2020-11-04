package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

type RoundRobinUrl struct {
	urls       []string
	currentUrl int
	state      chan bool
}

var zookeeper = "zookeeper:2181"
var nginx = "nginx:80"

func serveReverseProxy(w http.ResponseWriter, r *http.Request, targetUrl string) {
	url, _ := url.Parse("http://" + targetUrl)
	proxy := httputil.NewSingleHostReverseProxy(url)

	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host

	proxy.ServeHTTP(w, r)
}

func (urls *RoundRobinUrl) getGserveUrl() (currentUrl string, err error) {
	// todo: add mutex.lock()
	if len(urls.urls) > 0 {
		currentUrl = urls.urls[urls.currentUrl%len(urls.urls)]
		urls.currentUrl = (urls.currentUrl + 1) % len(urls.urls)
		// todo: add mutex.unlock()
	} else {
		return "", errors.New("Cant discover gserves")
	}
	return
}

func (urls *RoundRobinUrl) handleRequestAndRedirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	url, err := urls.getGserveUrl()
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	fmt.Println(url, urls)
	// w.Write([]byte(url))
	r.URL.Path = ""
	serveReverseProxy(w, r, url)
}

//todo refactor
func (roundRobin *RoundRobinUrl) discoverGserveNodes(conn *zk.Conn) {
	for {
		_, _, event, _ := conn.ChildrenW("/zookeeper")
		<-event
		nodes, _, _, _ := conn.ChildrenW("/zookeeper")
		roundRobin.urls = []string{}
		for _, v := range nodes {
			if v != "quota" {
				roundRobin.urls = append(roundRobin.urls, v)
				fmt.Println(nodes, roundRobin.urls)
				if len(roundRobin.urls) >= 2 {
					fmt.Println("url >= 2 :) ")
					roundRobin.state <- true
					close(roundRobin.state)
				}
			}
		}
	}
}

func (urls *RoundRobinUrl) runRouter() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveReverseProxy(w, r, nginx)
	})

	http.HandleFunc("/library", urls.handleRequestAndRedirect)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func main() {
	roundRobin := &RoundRobinUrl{state: make(chan bool)}
	fmt.Println("Starting gproxy..")
	conn, _, err := zk.Connect([]string{zookeeper}, time.Second*10) //*10)
	defer conn.Close()
	if err != nil {
		panic(err)
	}

	go roundRobin.discoverGserveNodes(conn)
	<-roundRobin.state
	time.Sleep(50)
	roundRobin.runRouter()
}
