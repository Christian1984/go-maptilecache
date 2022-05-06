package main

import (
	"fmt"
	"net/http"

	"github.com/Christian1984/go-maptilecache/cache"
)

func main() {
	httpListen := "0.0.0.0:9001"
	cache.New("osm", "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 1)

	fmt.Println("Map Tile Cache listening at " + httpListen)
	http.ListenAndServe(httpListen, nil)
}
