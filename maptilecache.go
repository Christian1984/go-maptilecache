package maptilecache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
)

const DEFAULT_HTTP_CLIENT_TIMEOUT = 6 * time.Second

type FilePath struct {
	Path     string
	FullPath string
}

type CacheStats struct {
	BytesServedFromCache  int
	BytesServedFromHDD    int
	BytesServedFromMemory int
	BytesServedFromOrigin int
}

type Cache struct {
	Port            string
	Route           []string
	RouteString     string
	UrlScheme       string
	StructureParams []string
	TimeToLive      time.Duration
	ForwardHeaders  bool
	SharedMemCache  *SharedMemoryCache
	Client          *http.Client
	ApiKey          string
	Stats           CacheStats
	Logger          LoggerConfig
}

type CacheConfig struct {
	Port              string
	Route             []string
	UrlScheme         string
	StructureParams   []string
	TimeToLive        time.Duration
	ForwardHeaders    bool
	SharedMemoryCache *SharedMemoryCache
	HttpClientTimeout time.Duration
	ApiKey            string
	DebugLogger       func(string)
	InfoLogger        func(string)
	WarnLogger        func(string)
	ErrorLogger       func(string)
	StatsLogDelay     time.Duration
}

func New(config CacheConfig) (*Cache, error) {
	start := time.Now()

	routeString := routeString(config.Route)

	timeout := config.HttpClientTimeout

	if config.HttpClientTimeout <= 0 {
		timeout = DEFAULT_HTTP_CLIENT_TIMEOUT
	}

	c := Cache{
		Port:            config.Port,
		Route:           config.Route,
		RouteString:     routeString,
		UrlScheme:       config.UrlScheme,
		StructureParams: config.StructureParams,
		TimeToLive:      config.TimeToLive,
		ForwardHeaders:  config.ForwardHeaders,
		SharedMemCache:  config.SharedMemoryCache,
		Client:          &http.Client{Timeout: timeout},
		ApiKey:          config.ApiKey,
		Logger: LoggerConfig{
			LogPrefix:     "Cache[" + routeString + "]",
			LogDebugFunc:  config.DebugLogger,
			LogInfoFunc:   config.InfoLogger,
			LogWarnFunc:   config.WarnLogger,
			LogErrorFunc:  config.ErrorLogger,
			StatsLogDelay: config.StatsLogDelay,
		},
	}

	c.logDebug("Timeout: " + timeout.String())

	if len(config.Route) < 1 {
		return &c, errors.New("could not initialize cache, reason: route invalid, must have at least one entry")
	}

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/"+routeString+"/", c.serve)
	go http.ListenAndServe("localhost:"+c.Port, serverMux)

	c.InitLogStatsRunner()

	duration := time.Since(start)
	c.logInfo("New Cache initialized on http://localhost:" + c.Port + "/" + routeString + "/ (took " + duration.String() + ")")

	return &c, nil

}

func routeString(route []string) string {
	return strings.Join(route, "/")
}

func (c *Cache) WipeCache() error {
	c.logInfo("Wiping cache...")

	start := time.Now()

	cacheRoot := filepath.Join(append([]string{"."}, c.Route...)...)
	if isPathDangerous(cacheRoot) {
		msg := "Cache could not be wiped, illegal cacheRoot: [" + cacheRoot + "]"
		c.logError(msg)
		return errors.New(msg)
	}

	err := os.RemoveAll(cacheRoot)

	if err != nil {
		c.logWarn("Cache could not be wiped, reason: " + err.Error())
	} else {
		duration := time.Since(start)
		c.logInfo("Cache successfully wiped! (took " + duration.String() + ")")
	}

	return err
}

func (c *Cache) memoryMapLoad(requestIdPrefix string, requestParams *url.Values, x string, y string, z string) (*[]byte, error) {
	start := time.Now()
	key := c.makeFilepath(requestParams, x, y, z).FullPath

	if c.SharedMemCache == nil {
		msg := "SharedMemoryCache not set, cannot load tile with key [" + key + "] from memory map."
		c.logDebug(requestIdPrefix + msg)
		return nil, errors.New(msg)
	}

	data, exists := c.SharedMemCache.MemoryMapRead(c.RouteString, key)

	duration := time.Since(start)

	if exists {
		c.logDebug(requestIdPrefix + "Loaded tile from the MemoryMap with key [" + key + "] (took " + duration.String() + ")")
		return data, nil
	} else {
		c.logDebug(requestIdPrefix + "Tile for key [" + key + "] not found in MemoryMap (took " + duration.String() + ")")
		return nil, errors.New("Tile for key [" + key + "] not found in MemoryMap.")
	}
}

func (c *Cache) memoryMapStore(requestIdPrefix string, requestParams *url.Values, x string, y string, z string, data *[]byte) {
	start := time.Now()
	key := c.makeFilepath(requestParams, x, y, z).FullPath

	if c.SharedMemCache == nil {
		msg := "SharedMemoryCache not set, cannot store tile with key [" + key + "] in memory map."
		c.logDebug(requestIdPrefix + msg)
		return
	}

	c.SharedMemCache.MemoryMapWrite(c.RouteString, key, data)

	duration := time.Since(start)
	c.logDebug(requestIdPrefix + "Tile with " + strconv.Itoa(len(*data)) + " Bytes successfully saved to the MemoryMap with key [" + key + "] (took " + duration.String() + ")")
}

func (c *Cache) isFileOutdated(modtime time.Time) bool {
	age := time.Now().Sub(modtime)
	return age > c.TimeToLive
}

func (c *Cache) ValidateCache() {
	c.logInfo("Validating cache...")

	start := time.Now()

	root := filepath.Join(append([]string{"."}, c.Route...)...)

	if _, statErr := os.Stat(root); statErr != nil {
		c.logDebug("Cache directory not yet created. Aborting cleanup!")
		return
	}

	var totalSize int64 = 0
	var removedFilesSize int64 = 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size := info.Size()
			infoString := fmt.Sprintf("Inspecting file [%s] => size: %d Bytes, modtime: %s", path, size, info.ModTime().String())
			c.logDebug(infoString)

			totalSize += size

			if c.isFileOutdated(info.ModTime()) {
				c.logDebug("[" + path + "] is outdated. Removing file from cache...")
				removeErr := os.Remove(path)

				if removeErr != nil {
					c.logWarn("Could not remove [" + path + "]")
					return nil
				}

				removedFilesSize += size

				c.logDebug("Removed file [" + path + "]")
			} else {
				c.logDebug("File [" + path + "] is current.")
			}
		} else {
			files, err := ioutil.ReadDir(path)
			if err == nil && len(files) == 0 {
				err = os.Remove(path)

				if err == nil {
					c.logDebug("Removed folder [" + path + "]")
				}
			}
		}

		return nil
	})

	if err != nil {
		c.logWarn("Could not clean cache, reason: " + err.Error())
		return
	}

	duration := time.Since(start)
	c.logInfo(fmt.Sprintf("Cache validated and cleaned! (Size before: %d Bytes, Size now: %d Bytes, %d Bytes removed, took %s)", totalSize, totalSize-removedFilesSize, removedFilesSize, duration.String()))
}

func (c *Cache) PreloadMemoryMap() {
	if c.SharedMemCache == nil {
		msg := "SharedMemoryCache not set, cannot preload memory map"
		c.logDebug(msg)
		return
	}

	c.logInfo("Preloading cached tiles into memory map...")

	start := time.Now()

	root := filepath.Join(append([]string{"."}, c.Route...)...)

	if _, statErr := os.Stat(root); statErr != nil {
		c.logDebug("Cache directory not yet created. Aborting preload!")
		return
	}

	var totalSize int64 = 0
	tilesStored := 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalSize += info.Size()
			data, err := ioutil.ReadFile(path)

			if err != nil {
				c.logWarn("Could not preload file " + path + ", reason: " + err.Error())
			} else {
				if c.SharedMemCache.MaxSizeReachedMutex() {
					return errors.New("SharedMemoryCache exceeded its max size during preload... Preload aborted after " + strconv.Itoa(tilesStored) + " tiles.")
				}

				c.SharedMemCache.MemoryMapWrite(c.RouteString, path, &data)
				tilesStored++
				c.logDebug("Preloaded " + strconv.Itoa(len(data)) + " bytes from file " + path + " into MemoryMap [" + c.RouteString + "] with tileKey [" + path + "].")
			}
		}

		return nil
	})

	if err != nil {
		c.logWarn("Could not perform preload, reason: " + err.Error())
		return
	}

	duration := time.Since(start)
	c.logInfo(fmt.Sprintf("Cache data preloaded into memory! %d Bytes loaded, %d tiles stored, took %s)", totalSize, tilesStored, duration.String()))
}

func (c *Cache) request(requestIdPrefix string, x string, y string, z string, s string, params *url.Values, sourceHeader *http.Header) (*[]byte, error) {
	start := time.Now()

	url := c.UrlScheme
	url = strings.Replace(url, "{s}", s, 1)
	url = strings.Replace(url, "{x}", x, 1)
	url = strings.Replace(url, "{y}", y, 1)
	url = strings.Replace(url, "{z}", z, 1)

	if strings.Contains(c.UrlScheme, "{apiKey}") && strings.TrimSpace(c.ApiKey) == "" {
		c.logWarn("Trying to replace {apiKey}, but ApiKey is not configured!")
	}
	url = strings.Replace(url, "{apiKey}", c.ApiKey, 1)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logError("Could not create request, reason: " + err.Error())
		return nil, err
	}

	if c.ForwardHeaders {
		req.Header = *sourceHeader
	}

	query := req.URL.Query()
	if params != nil {
		for key, values := range *params {
			for _, value := range values {
				query.Add(key, value)
			}
		}
	}

	req.URL.RawQuery = query.Encode()

	c.logDebug(requestIdPrefix + "Requesting tile from " + req.URL.RequestURI())
	c.logDebug(requestIdPrefix + fmt.Sprintf("Request Headers: %s", req.Header))

	requestStart := time.Now()
	c.logDebug(requestIdPrefix + "Starting request at [" + requestStart.String() + "]")

	resp, err := c.Client.Do(req)

	requestDuration := time.Since(requestStart)
	c.logDebug(requestIdPrefix + "Request from [" + requestStart.String() + "] finished (took " + requestDuration.String() + ").")

	if err != nil {
		c.logError("Could not request tile, reason: " + err.Error())
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		c.logError("Could not request tile, bad status code: " + strconv.Itoa(resp.StatusCode))
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		c.logError("Could parse response body, reason: " + err.Error())
		return nil, err
	}

	c.logDebug(requestIdPrefix + "Received " + strconv.Itoa(len(bodyBytes)) + " Bytes from " + url)

	if !c.isValidTile(requestIdPrefix, &bodyBytes) {
		length := len(bodyBytes)
		if length > 20 {
			length = 20
		}

		c.logDebug(requestIdPrefix + "Invalid response body received. First " + strconv.Itoa(length) + " bytes received: " + string(bodyBytes[:length+1]))

		return nil, errors.New("Invalid response body received.")
	}

	go c.save(requestIdPrefix, params, x, y, z, &bodyBytes)
	go c.memoryMapStore(requestIdPrefix, params, x, y, z, &bodyBytes)

	duration := time.Since(start)
	c.logDebug(requestIdPrefix + "Serving " + strconv.Itoa(len(bodyBytes)) + " Bytes to client (took " + duration.String() + ")")

	return &bodyBytes, nil
}

func (c *Cache) makeFilepath(requestParams *url.Values, x string, y string, z string) FilePath {
	pathArray := append([]string{"."}, c.Route...)

	var additionalSubfolders []string
	for _, requiredKey := range c.StructureParams {
		value := strings.TrimSpace(requestParams.Get(requiredKey))

		if len(value) > 0 {
			m1 := regexp.MustCompile(`[<>:"\/\\|?*]`)
			value = m1.ReplaceAllString(value, "-")
			additionalSubfolders = append(additionalSubfolders, value)
		}
	}

	pathArray = append(pathArray, additionalSubfolders...)
	pathArray = append(pathArray, z, y)

	path := filepath.Join(pathArray...)
	fullPath := filepath.Join(path, x+".png")

	return FilePath{
		Path:     path,
		FullPath: fullPath,
	}
}

func isPathDangerous(path string) bool {
	trimmedPath := strings.TrimSpace(path)
	return trimmedPath == "" || trimmedPath == "/" || trimmedPath == "C:\\"
}

func (c *Cache) load(requestIdPrefix string, requestParams *url.Values, x string, y string, z string) (*[]byte, error) {
	start := time.Now()

	fp := c.makeFilepath(requestParams, x, y, z)
	data, err := ioutil.ReadFile(fp.FullPath)

	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, errors.New("File empty!")
	}

	t, err := times.Stat(fp.FullPath)

	if err != nil {
		return nil, err
	}

	c.logDebug(requestIdPrefix + "ModTime for " + fp.FullPath + ": " + t.ModTime().String())

	if c.isFileOutdated(t.ModTime()) {
		return nil, errors.New("Tile is too old!")
	}

	duration := time.Since(start)
	c.logDebug(requestIdPrefix + "Loaded tile from " + fp.FullPath + " (took " + duration.String() + ")")

	return &data, nil
}

func (c *Cache) isValidTile(requestIdPrefix string, bytes *[]byte) bool {
	if len(*bytes) < 4 {
		c.logDebug(requestIdPrefix + "Tile invalid, response body was empty.")
		return false
	}

	header := strings.ToLower(string((*bytes)[1:4]))
	if header != "png" {
		c.logDebug(requestIdPrefix + "Tile invalid, header != [PNG], got [" + header + "] instead")
		return false
	}

	return true
}

func (c *Cache) save(requestIdPrefix string, requestParams *url.Values, x string, y string, z string, data *[]byte) error {
	start := time.Now()

	fp := c.makeFilepath(requestParams, x, y, z)

	c.logDebug(requestIdPrefix + "Saving " + strconv.Itoa(len(*data)) + " Bytes to filesystem at " + fp.FullPath)

	dirErr := os.MkdirAll(fp.Path, os.ModePerm)

	if dirErr != nil {
		c.logError("Could not save tile, reason: " + dirErr.Error())
		return dirErr
	}
	fileErr := ioutil.WriteFile(fp.FullPath, *data, 0644)

	if fileErr != nil {
		c.logError("Could not save tile, reason: " + fileErr.Error())
		return fileErr
	}

	duration := time.Since(start)
	c.logDebug(requestIdPrefix + "Tile with " + strconv.Itoa(len(*data)) + " Bytes successfully saved to " + fp.FullPath + " (took " + duration.String() + ")")
	return nil
}

func (c *Cache) serve(w http.ResponseWriter, req *http.Request) {
	// route format: /{route}/{s}/{z}/{y}/{x}/?params
	start := time.Now()

	randId := rand.Int63n(256 * 256 * 256 * 256)
	randIdString := fmt.Sprintf("%08X", randId)
	requestIdPrefix := "[reqID " + randIdString + "] "

	c.logDebug(requestIdPrefix + "Received request with RequestURI [" + req.RequestURI + "]")

	// c.logDebug(requestIdPrefix + "Enter Sleep")
	// time.Sleep(3 * time.Second)
	// c.logDebug(requestIdPrefix + "Sleep done!")

	requestPath := strings.Split(req.URL.Path, "/")

	if len(requestPath) < 5+len(c.Route) {
		c.logError("Bad Request: Not enough arguments in route [" + req.RequestURI + "]")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return
	}

	s := requestPath[1+len(c.Route)]
	z := requestPath[2+len(c.Route)]
	y := requestPath[3+len(c.Route)]
	x := requestPath[4+len(c.Route)]

	c.logDebug(requestIdPrefix + "Params found in route: s=[" + s + "], x=[" + x + "], y=[" + y + "], z=[" + z + "]")

	params := req.URL.Query()
	c.logDebug(requestIdPrefix + "Request params found : " + fmt.Sprint(params))

	var data *[]byte
	var err error

	//c.logDebug(requestIdPrefix + "Trying to load tile, total numbers tiles in this cache's memory map: " + strconv.Itoa(len(*(c.SharedMemCache.MemoryMaps)[c.RouteString].Tiles)))
	data, err = c.memoryMapLoad(requestIdPrefix, &params, x, y, z)

	if err != nil || data == nil {
		c.logDebug(requestIdPrefix + "Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from MemoryMap, will try HDD...")
		data, err = c.load(requestIdPrefix, &params, x, y, z)

		if err != nil || data == nil {
			c.logDebug(requestIdPrefix + "Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from HDD, will request it from server...")
		} else {
			c.logDebug(requestIdPrefix + "Tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] found in HDD-Storage!")
			c.Stats.BytesServedFromHDD += len(*data)
			c.memoryMapStore(requestIdPrefix, &params, x, y, z, data)
		}
	} else {
		c.logDebug(requestIdPrefix + "Tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] found in MemoryMap!")
		c.Stats.BytesServedFromMemory += len(*data)
	}

	if err != nil || data == nil {
		errString := "data == nil"

		if err != nil {
			errString = err.Error()
		}

		c.logDebug(requestIdPrefix + "Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "], reason: " + errString)
		c.logDebug(requestIdPrefix + "Sending request to server...")

		sourceHeader := req.Header.Clone()

		data, err = c.request(requestIdPrefix, x, y, z, s, &params, &sourceHeader)

		if err != nil || data == nil {
			c.logWarn("Could not fetch tile for x=[" + x + "], y=[" + y + "], z=[" + z + "].")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
			return
		} else {
			c.logDebug(requestIdPrefix + "Fetched tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from server (" + strconv.Itoa(len(*data)) + " Bytes)!")
			c.Stats.BytesServedFromOrigin += len(*data)
		}
	} else {
		c.logDebug(requestIdPrefix + "Loaded tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from cache (" + strconv.Itoa(len(*data)) + " Bytes)!")
		c.Stats.BytesServedFromCache += len(*data)
	}

	//c.LogStats()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(*data)))

	duration := time.Since(start)
	c.logDebug(requestIdPrefix + "Processing request with RequestURI: [" + req.RequestURI + "] took " + duration.String())

	w.Write(*data)
}
