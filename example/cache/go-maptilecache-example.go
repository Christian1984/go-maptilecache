package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/Christian1984/go-maptilecache"
)

func sendTestRequest(url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Could not create request for url " + url + ", reason: " + err.Error())
		return
	}

	req.Header.Set("User-Agent", "curl/7.54.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "*")

	client := &http.Client{}
	client.Do(req)
}

func testOne() {
	sendTestRequest("http://localhost:9001/maptilecache/osm/a/6/10/10/")
}

func testOneWithParams() {
	sendTestRequest("http://localhost:9001/maptilecache/osm/a/6/10/10/?test=ok")
}

func testMany(n int) {
	for i := 0; i < n; i++ {
		x := strconv.Itoa(rand.Intn(5))
		y := strconv.Itoa(rand.Intn(5))
		z := strconv.Itoa(rand.Intn(4) + 3)

		sendTestRequest("http://localhost:9001/maptilecache/osm/a/" + z + "/" + y + "/" + x + "/")

		time.Sleep(1 * time.Second)
	}
}

func initTestEndpoint() {
	http.HandleFunc("/test/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		w.Write([]byte("Done"))
	})
	maptilecache.New([]string{"maptilecache", "test"}, "http://localhost:9001/test/", 20*time.Second, "")
}

func main() {
	httpListen := "0.0.0.0:9001"

	maptilecache.New([]string{"maptilecache", "osm"}, "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 90*24*time.Hour, "")
	maptilecache.New([]string{"maptilecache", "otm"}, "https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png", 90*24*time.Hour, "")

	maptilecache.New([]string{"maptilecache", "oaip"}, "http://{s}.tile.maps.openaip.net/geowebcache/service/tms/1.0.0/openaip_basemap@EPSG%3A900913@png/{z}/{x}/{y}.png", 90*24*time.Hour, "")

	go http.ListenAndServe(httpListen, nil)
	fmt.Println("Map Tile Cache listening at " + httpListen)

	time.Sleep(1 * time.Second)
	//testOne()
	//testOneWithParams()
	//testMany(20)

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
