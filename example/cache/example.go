package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/Christian1984/go-maptilecache"
	"github.com/Christian1984/go-maptilecache/example/cache/secrets"
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

/*
func initTestEndpoint() {
	http.HandleFunc("/test/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Header)
		w.Write([]byte("Done"))
	})
	maptilecache.New([]string{"maptilecache", "test"}, "http://localhost:9001/test/", []string{}, 20*time.Second, "",
		maptilecache.PrintlnDebugLogger, maptilecache.PrintlnInfoLogger, maptilecache.PrintlnWarnLogger, maptilecache.PrintlnErrorLogger, 0)
}
*/

func main() {
	httpListen := "0.0.0.0:9001"
	statsLogDelay := 0 * time.Second

	maxMemoryFootprint := 1024 * 1024 * 256 // 256 MB
	ttl := 10 * 24 * time.Hour

	sharedmemorycache := maptilecache.NewSharedMemoryCache(maxMemoryFootprint, maptilecache.PrintlnDebugLogger, maptilecache.PrintlnInfoLogger, maptilecache.PrintlnWarnLogger, maptilecache.PrintlnErrorLogger)

	osmcache, err := maptilecache.New(
		[]string{"maptilecache", "osm"},
		"http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
		[]string{},
		ttl,
		true,
		sharedmemorycache,
		"",
		maptilecache.PrintlnDebugLogger,
		maptilecache.PrintlnInfoLogger,
		maptilecache.PrintlnWarnLogger,
		maptilecache.PrintlnErrorLogger,
		statsLogDelay,
	)

	if err == nil {
		osmcache.ValidateCache()
		osmcache.PreloadMemoryMap()
	}

	otmcache, err := maptilecache.New(
		[]string{"maptilecache", "otm"},
		"https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png",
		[]string{},
		90*24*time.Hour,
		true,
		sharedmemorycache,
		"",
		maptilecache.PrintlnDebugLogger,
		maptilecache.PrintlnInfoLogger,
		maptilecache.PrintlnWarnLogger,
		maptilecache.PrintlnErrorLogger,
		statsLogDelay,
	)

	if err == nil {
		otmcache.ValidateCache()
		otmcache.PreloadMemoryMap()
	}

	maptilecache.New(
		[]string{"maptilecache", "oaip-airports"},
		"https://api.tiles.openaip.net/api/data/airports/{z}/{x}/{y}.png?apiKey={apiKey}",
		[]string{},
		90*24*time.Hour,
		false,
		sharedmemorycache,
		secrets.OAPI_API_KEY,
		maptilecache.PrintlnDebugLogger,
		maptilecache.PrintlnInfoLogger,
		maptilecache.PrintlnWarnLogger,
		maptilecache.PrintlnErrorLogger,
		statsLogDelay,
	)

	maptilecache.New(
		[]string{"maptilecache", "oaip-airspaces"},
		"https://api.tiles.openaip.net/api/data/airspaces/{z}/{x}/{y}.png?apiKey={apiKey}",
		[]string{},
		90*24*time.Hour,
		false,
		sharedmemorycache,
		secrets.OAPI_API_KEY,
		maptilecache.PrintlnDebugLogger,
		maptilecache.PrintlnInfoLogger,
		maptilecache.PrintlnWarnLogger,
		maptilecache.PrintlnErrorLogger,
		statsLogDelay,
	)

	/*
		maptilecache.New([]string{"maptilecache", "ofm"}, "https://nwy-tiles-api.prod.newaydata.com/tiles/{z}/{x}/{y}.png", []string{"path"}, 90*24*time.Hour, true, maxMemoryFootprint, "",
			maptilecache.PrintlnDebugLogger, maptilecache.PrintlnInfoLogger, maptilecache.PrintlnWarnLogger, maptilecache.PrintlnErrorLogger, statsLogDelay)
	*/

	go http.ListenAndServe(httpListen, nil)
	fmt.Println("Map Tile Cache listening at " + httpListen)

	time.Sleep(1 * time.Second)
	//testOne()
	//testOneWithParams()
	//testMany(20)

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
