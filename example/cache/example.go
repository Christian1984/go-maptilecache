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
	statsLogDelay := 0 * time.Second

	ttl := 10 * 24 * time.Hour
	maxMemoryFootprint := 1024 * 1024 * 4 // 4 MB
	//maxMemoryFootprint := 1024 * 1024 * 256 // 256 MB

	sharedMemoryCacheConfig := maptilecache.SharedMemoryCacheConfig{
		MaxSizeBytes:          maxMemoryFootprint,
		EnsureMaxSizeInterval: 10 * time.Second,
		DebugLogger:           maptilecache.PrintlnDebugLogger,
		InfoLogger:            maptilecache.PrintlnInfoLogger,
		WarnLogger:            maptilecache.PrintlnWarnLogger,
		ErrorLogger:           maptilecache.PrintlnErrorLogger,
	}
	sharedMemoryCache := maptilecache.NewSharedMemoryCache(sharedMemoryCacheConfig)
	// var sharedMemoryCache *maptilecache.SharedMemoryCache = nil

	osmCacheConfig := maptilecache.CacheConfig{
		Port:              "9002",
		Route:             []string{"maptilecache", "osm"},
		UrlScheme:         "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
		TimeToLive:        ttl,
		ForwardHeaders:    true,
		SharedMemoryCache: sharedMemoryCache,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     statsLogDelay,
	}
	osmCache, err := maptilecache.New(osmCacheConfig)

	if err == nil {
		osmCache.ValidateCache()
		osmCache.PreloadMemoryMap()
	}

	otmCacheConfig := maptilecache.CacheConfig{
		Port:              "9003",
		Route:             []string{"maptilecache", "otm"},
		UrlScheme:         "https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png",
		TimeToLive:        ttl,
		ForwardHeaders:    true,
		SharedMemoryCache: sharedMemoryCache,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     statsLogDelay,
	}
	otmcache, err := maptilecache.New(otmCacheConfig)

	if err == nil {
		otmcache.ValidateCache()
		otmcache.PreloadMemoryMap()
	}

	oaipAirportsCacheConfig := maptilecache.CacheConfig{
		Port:              "9004",
		Route:             []string{"maptilecache", "oaip-airports"},
		UrlScheme:         "https://api.tiles.openaip.net/api/data/airports/{z}/{x}/{y}.png?apiKey={apiKey}",
		TimeToLive:        ttl,
		ForwardHeaders:    false, // required for openAIP
		SharedMemoryCache: sharedMemoryCache,
		ApiKey:            secrets.OAPI_API_KEY,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     statsLogDelay,
	}
	maptilecache.New(oaipAirportsCacheConfig)

	oaipAirspacesCacheConfig := maptilecache.CacheConfig{
		Port:              "9005",
		Route:             []string{"maptilecache", "oaip-airspaces"},
		UrlScheme:         "https://api.tiles.openaip.net/api/data/airspaces/{z}/{x}/{y}.png?apiKey={apiKey}",
		TimeToLive:        ttl,
		ForwardHeaders:    false, // required for openAIP
		SharedMemoryCache: sharedMemoryCache,
		ApiKey:            secrets.OAPI_API_KEY,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     statsLogDelay,
	}
	maptilecache.New(oaipAirspacesCacheConfig)

	oaipNavaidsCacheConfig := maptilecache.CacheConfig{
		Port:              "9006",
		Route:             []string{"maptilecache", "oaip-navaids"},
		UrlScheme:         "https://api.tiles.openaip.net/api/data/navaids/{z}/{x}/{y}.png?apiKey={apiKey}",
		TimeToLive:        ttl,
		ForwardHeaders:    false, // required for openAIP
		SharedMemoryCache: sharedMemoryCache,
		ApiKey:            secrets.OAPI_API_KEY,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     statsLogDelay,
	}
	maptilecache.New(oaipNavaidsCacheConfig)

	time.Sleep(1 * time.Second)
	//testOne()
	//testOneWithParams()
	//testMany(20)

	fmt.Println("Press Enter Key to quit")
	fmt.Scanln()
}
