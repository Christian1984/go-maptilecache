package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Christian1984/go-maptilecache/cache"
)

func testRequest() {
	//http.Get("http://localhost:9001/maptilecache/test/a/6/10/10/")
	http.Get("http://localhost:9001/maptilecache/osm/a/6/11/11/")
}

func main() {
	httpListen := "0.0.0.0:9001"
	cache.New([]string{"maptilecache", "test"}, "http://localhost:9001/test/", 1*time.Hour, "", "go-maptilecache-test")
	cache.New([]string{"maptilecache", "osm"}, "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 1*time.Hour, "", "go-maptilecache-osm")

	fmt.Println("Map Tile Cache listening at " + httpListen)

	http.HandleFunc("/test/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		w.Write([]byte("Done"))
	})

	go http.ListenAndServe(httpListen, nil)
	time.Sleep(1 * time.Second)
	testRequest()

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
