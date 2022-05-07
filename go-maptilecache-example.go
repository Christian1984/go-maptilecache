package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Christian1984/go-maptilecache/cache"
)

func testRequest() {
	//http.Get("http://localhost:9001/maptilecache/test/a/6/10/10/")

	/*for i := 0; i < 100; i++ {
		x := strconv.Itoa(rand.Intn(5))
		y := strconv.Itoa(rand.Intn(5))
		z := strconv.Itoa(rand.Intn(4) + 3)

		http.Get("http://localhost:9001/maptilecache/osm/a/" + z + "/" + y + "/" + x + "/")

		time.Sleep(1 * time.Second)
	}*/
}

func main() {
	httpListen := "0.0.0.0:9001"
	cache.New([]string{"maptilecache", "osm"}, "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 30*24*time.Hour, "", "go-maptilecache-osm")

	/*
		http.HandleFunc("/test/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.Header)
			w.Write([]byte("Done"))
		})
		cache.New([]string{"maptilecache", "test"}, "http://localhost:9001/test/", 20*time.Second, "", "go-maptilecache-test")
	*/

	go http.ListenAndServe(httpListen, nil)
	fmt.Println("Map Tile Cache listening at " + httpListen)

	time.Sleep(1 * time.Second)
	testRequest()

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
