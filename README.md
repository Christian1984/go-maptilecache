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
const osm = new L.TileLayer("http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
    format: "image/png",
    subdomains: ["a", "b", "c"]
});
```

you would now point it at your go-maptilecache instance by configuring it like this

```
const osm = new L.TileLayer("http://localhost:9001/maptilecache/osm/{s}/{z}/{y}/{x}/", {
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
    maptilecache.New([]string{"maptilecache", "osm"}, "http://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", 90*24*time.Hour, "")
    http.ListenAndServe("0.0.0.0:9001", nil)
}

```

# Examples

See [here](https://github.com/Christian1984/go-maptilecache/tree/master/example) for examples.

# TODO:

- api key usage is not yet implemented
- query params are not yet forwarded