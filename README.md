This project is a simple, open source map tile cache written go and licensed under the MIT license.

Upon receiving a map tile request the cache checks for cached map tiles and evaluates its age against a previously configured "Time to Live".

- If the tile exists and is still current it will be served directly from the cache.
- Otherwise it will be fetched from the remote server of the map tile provider, served to the client while the cache will be updated accordingly.

# Benefits

Caching map tiles locally instead of fetching them remotely reduces

- network traffic
- server load
- potential costs when using metered APIs

Plus, it allows you to hide your API-Keys on the server side instead of distributing them to the client in your JavaScript.

# Usage

Instead of configuring your mapping library (like leaflet, for example) to fetch tiles directly from the map tile provider, route it to the server that is running the go-maptilecache instance.

For example, if you had previously configured a leaflet tile layer in your JavaScript like this

```
const osm = new L.TileLayer("https://api.tiles.openaip.net/api/data/airports/{z}/{x}/{y}.png?apiKey=<...>", {
    format: "image/png",
    subdomains: ["a", "b", "c"]
});
```

you would now point it at your go-maptilecache instance by configuring it like this

```
const osm = new L.TileLayer("http://localhost:9001/maptilecache/oaip-airports/{s}/{z}/{y}/{x}/", {
    format: "image/png",
    subdomains: ["a", "b", "c"]
});
```

to send all leaflet tile requests to the cache instead of the server where the tiles are hosted initially. 

Then configure and spin up the cache with

```
package main

import (
    "net/http"
    "time"

    "github.com/Christian1984/go-maptilecache"
)

func main() {
    // create an in-memory cache that can be shared across multiple maptile caches
	sharedMemoryCacheConfig := maptilecache.SharedMemoryCacheConfig{
		MaxSizeBytes:          256 * 1024 * 1024,
		EnsureMaxSizeInterval: 10 * time.Second,
		DebugLogger:           maptilecache.PrintlnDebugLogger,
		InfoLogger:            maptilecache.PrintlnInfoLogger,
		WarnLogger:            maptilecache.PrintlnWarnLogger,
		ErrorLogger:           maptilecache.PrintlnErrorLogger,
	}
	sharedMemoryCache := maptilecache.NewSharedMemoryCache(sharedMemoryCacheConfig)

    // create one or more maptile caches
	oaipAirportsCacheConfig := maptilecache.CacheConfig{
		Route:             []string{"maptilecache", "oaip-airports"},
		UrlScheme:         "https://api.tiles.openaip.net/api/data/airports/{z}/{x}/{y}.png?apiKey={apiKey}",
		TimeToLive:        10 * 24 * time.Hour,
		ForwardHeaders:    false, // required for openAIP, default treu works for most APIs
		SharedMemoryCache: sharedMemoryCache,
		ApiKey:            secrets.OAPI_API_KEY,
		DebugLogger:       maptilecache.PrintlnDebugLogger,
		InfoLogger:        maptilecache.PrintlnInfoLogger,
		WarnLogger:        maptilecache.PrintlnWarnLogger,
		ErrorLogger:       maptilecache.PrintlnErrorLogger,
		StatsLogDelay:     1 * time.Hour,
	}
	maptilecache.New(oaipAirportsCacheConfig)
    http.ListenAndServe("localhost:9001", nil)
}

```

# Headers and Request Params

Both headers and request parameters will be forwarded to the server "as is".

# Organizing The Cache With "Params-Based" Subfolders

The constructor parameter `structureParams` allows you to specify parameter keys that are expected by the cache and that can be used to create subfolders inside the cache root directory. As an example, consider openflight maps.

OFM requires client to add a request parameter `path` which identifies the current AIRAC cycle (aviation term). A request url for Leaflet to request tiles from OFM would look like this:

```
https://nwy-tiles-api.prod.newaydata.com/tiles/{z}/{x}/{y}.png?path=2200/aero/latest
```

and the corresponding request to the cache would be

```
http://localhost:9001/maptilecache/ofm/{s}/{z}/{y}/{x}/?path=2200/aero/latest
```

Now configure your cache like this

```
ofmCacheConfig := maptilecache.CacheConfig{
	Route:             []string{"maptilecache", "ofm"},
	UrlScheme:         "https://nwy-tiles-api.prod.newaydata.com/tiles/{z}/{x}/{y}.png",
    StructureParams:   []string{"path"}, // <= expected query parameters
    /* ... */
}
maptilecache.New(ofmCacheConfig)
```

The cache will forward the `path` param to the server and will organize all locally cached tiles according to the AIRAC cycle. The folder structure would then look like this: `maptilecache/ofm/{AIRAC-cycle}/z/y/x.png`

# Examples

See [here](https://github.com/Christian1984/go-maptilecache/tree/master/example) for examples.