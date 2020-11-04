package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/samuel/go-zookeeper/zk"
)

type Gserve struct {
	hostname  string
	hbase     string
	zookeeper string
	hostport  string
}

type Message struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func putBook(unencodedBook []byte, hostAddress string) *http.Response {
	var unencodedRows RowsType
	json.Unmarshal(unencodedBook, &unencodedRows)
	encodedRows := unencodedRows.encode()
	encodedJSON, _ := json.Marshal(encodedRows)
	res, _ := http.Post(hostAddress+"se2:library/fakerow", "application/json", bytes.NewBuffer(encodedJSON))
	fmt.Println(res.Status)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	return res
}

func getScanner(hostAddress string) string {
	req, _ := http.NewRequest("PUT", hostAddress+"se2:library/scanner", bytes.NewBuffer([]byte("<Scanner batch=\"100\"/>")))
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	res, _ := client.Do(req)
	log.Println("Res with scanner: " + res.Header.Get("Location"))
	return res.Header.Get("Location")
}

func getLibrary(hostAddress string) RowsType {
	log.Println("Starting getLibrary " + hostAddress)
	scanner := getScanner(hostAddress)
	req, _ := http.NewRequest("GET", scanner, nil)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	res, _ := client.Do(req)
	log.Println(res.Status)

	var library EncRowsType
	_ = json.NewDecoder(res.Body).Decode(&library)
	decodedJSON, _ := library.decode()
	byteArray, _ := json.Marshal(&decodedJSON)
	log.Println(string(byteArray))
	return decodedJSON
}

func Respond(w http.ResponseWriter, data Message) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (gserve *Gserve) addBook(w http.ResponseWriter, r *http.Request) {
	byteArr, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Respond(w, Message{false, "Invalid request"})
		return
	}
	res := putBook(byteArr, gserve.hbase)
	fmt.Fprintf(w, "POST %v+", res)
	fmt.Fprintf(w, "proudly served by "+gserve.hostname)
}

// func filterColumnNames(library *RowsType) {
// 	for i, row := range (*library).Row {
// 		for j, cell := range row.Cell {
// 			(*library).Row[i].Cell[j].Column = strings.SplitAfter(cell.Column, ":")[1]
// 		}
// 	}
// }
func (gserve *Gserve) getBooks(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting getBooks")
	library := getLibrary(gserve.hbase)
	log.Println("receive library")
	// w.Header().Set("Content-Type", "application/json")
	// filterColumnNames(&library)
	// w.Write(library)
	t := template.Must(template.ParseFiles("library.html"))
	var templateData = struct {
		Library  RowsType
		Hostname string
	}{
		library,
		gserve.hostname,
	}
	t.Execute(w, templateData)
}

func main() {
	log.Println("Starting gserve..")
	// hostName, _ := os.LookupEnv("host_name")
	hostName := os.Getenv("host_name")
	// hostName := "0.0.0.0"

	gserve := &Gserve{
		hostname:  hostName,
		hbase:     "http://hbase:8080/",
		zookeeper: "zookeeper:2181",
		hostport:  ":8888",
	}
	fmt.Println("starting gserve..")
	znodePath := "/zookeeper/" + gserve.hostname + gserve.hostport
	conn, _, err := zk.Connect([]string{gserve.zookeeper}, time.Second*10) //*10)
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	isExists, _, _ := conn.Exists(znodePath)
	if !isExists {
		path, err := conn.Create(znodePath, []byte(gserve.hostname+gserve.hostport), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
		if err != nil {
			panic(err)
		}
		fmt.Println("created ephemeral node: " + path)
	}
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", gserve.addBook).Methods("POST")
	router.HandleFunc("/", gserve.getBooks).Methods("GET")
	log.Fatal(http.ListenAndServe(gserve.hostport, router))
}
