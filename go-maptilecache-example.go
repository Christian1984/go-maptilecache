package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Christian1984/go-maptilecache/cache"
)

func testRequest() {
	http.Get("http://localhost:9001/maptilecache/osm/a/6/10/10/")
}

func main() {
	httpListen := "0.0.0.0:9001"
	cache.New([]string{"maptilecache", "osm"}, "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 1*time.Hour, "")

	fmt.Println("Map Tile Cache listening at " + httpListen)
	go http.ListenAndServe(httpListen, nil)
	time.Sleep(5 * time.Second)
	testRequest()

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
