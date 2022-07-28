package maptilecache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/djherbis/times"
)

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
	Route           []string
	UrlScheme       string
	StructureParams []string
	TimeToLive      time.Duration
	MemoryMap       map[string][]byte
	MemoryMapMutex  *sync.RWMutex
	MemoryMapSize   int
	ApiKey          string
	Stats           CacheStats
	Logger          LoggerConfig
}

func New(route []string,
	urlScheme string,
	structureParams []string,
	TimeToLiveDays time.Duration,
	maxMemoryFootprint int,
	apiKey string,
	debugLogger func(string),
	infoLogger func(string),
	warnLogger func(string),
	errorLogger func(string),
	statsLogDelay time.Duration) (Cache, error) {
	start := time.Now()

	c := Cache{
		Route:           route,
		UrlScheme:       urlScheme,
		StructureParams: structureParams,
		TimeToLive:      TimeToLiveDays,
		MemoryMap:       make(map[string][]byte),
		MemoryMapMutex:  &sync.RWMutex{},
		ApiKey:          apiKey,
		Logger: LoggerConfig{
			LogPrefix:     "Cache[" + strings.Join(route, "/") + "]",
			LogDebugFunc:  debugLogger,
			LogInfoFunc:   infoLogger,
			LogWarnFunc:   warnLogger,
			LogErrorFunc:  errorLogger,
			StatsLogDelay: statsLogDelay,
		},
	}

	if len(route) < 1 {
		return c, errors.New("Could not initialize Cache, reason: Route invalid, must have at least one entry!")
	}

	routeString := strings.Join(route, "/")

	http.HandleFunc("/"+routeString+"/", c.serve)

	c.InitLogStatsRunner()

	duration := time.Since(start)
	c.logInfo("New Cache initialized on route /" + routeString + "/ (took " + duration.String() + ")")

	return c, nil

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

func (c *Cache) memoryMapLoad(requestParams *url.Values, x string, y string, z string) ([]byte, error) {
	start := time.Now()
	key := c.makeFilepath(requestParams, x, y, z).FullPath

	c.MemoryMapMutex.RLock()
	data, exists := c.MemoryMap[key]
	c.MemoryMapMutex.RUnlock()

	duration := time.Since(start)

	if exists {
		c.logDebug("Loaded tile from the MemoryMap with key [" + key + "] (took " + duration.String() + ")")
		return data, nil
	} else {
		c.logDebug("Tile for key [" + key + "] not found in MemoryMap (took " + duration.String() + ")")
		return nil, errors.New("Tile for key [" + key + "] not found in MemoryMap.")
	}
}

func (c *Cache) memoryMapStore(requestParams *url.Values, x string, y string, z string, data *[]byte) {
	start := time.Now()
	key := c.makeFilepath(requestParams, x, y, z).FullPath

	c.MemoryMapMutex.Lock()
	c.MemoryMap[key] = *data
	c.MemoryMapMutex.Unlock()

	c.MemoryMapSize += len(*data) // TODO: check if existed previously

	// TODO: push key to history array, check size

	duration := time.Since(start)
	c.logDebug("Tile with " + strconv.Itoa(len(*data)) + " Bytes successfully saved to the MemoryMap with key [" + key + "] (took " + duration.String() + ")")
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

func (c *Cache) LoadMemoryMap() {
	c.logInfo("Preloading cached tiles into memory map...")

	start := time.Now()

	root := filepath.Join(append([]string{"."}, c.Route...)...)

	if _, statErr := os.Stat(root); statErr != nil {
		c.logDebug("Cache directory not yet created. Aborting preload!")
		return
	}

	var totalSize int64 = 0

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalSize += info.Size()

			// TODO: skip if size limit was exceeded

			data, err := ioutil.ReadFile(path)

			if err != nil {
				c.logWarn("Could not preload file " + path + ", reason: " + err.Error())
			} else {
				c.MemoryMapMutex.Lock()
				c.MemoryMap[path] = data
				c.MemoryMapMutex.Unlock()

				c.MemoryMapSize += len(data) // TODO: check if existed previously
				c.logDebug("Preloaded " + strconv.Itoa(len(data)) + " bytes from file " + path + " into MemoryMap.")
			}
		}

		return nil
	})

	if err != nil {
		c.logWarn("Could not perform preload, reason: " + err.Error())
		return
	}

	duration := time.Since(start)
	c.logInfo(fmt.Sprintf("Cache data preloaded into memory! %d Bytes loaded, took %s)", totalSize, duration.String()))
}

func (c *Cache) request(x string, y string, z string, s string, params *url.Values, sourceHeader *http.Header) ([]byte, error) {
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

	req.Header = *sourceHeader

	query := req.URL.Query()
	if params != nil {
		for key, values := range *params {
			for _, value := range values {
				query.Add(key, value)
			}
		}
	}

	req.URL.RawQuery = query.Encode()

	c.logDebug("Requesting tile from " + req.URL.RequestURI())
	c.logDebug(fmt.Sprintf("Request Headers: %s", req.Header))

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		c.logError("Could not request tile, reason: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logError("Could not request tile, bad status code: " + strconv.Itoa(resp.StatusCode))
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		c.logError("Could parse response body, reason: " + err.Error())
		return nil, err
	}

	c.logDebug("Received " + strconv.Itoa(len(bodyBytes)) + " Bytes from " + url)

	if len(bodyBytes) == 0 {
		c.logError("Invalid response body, reason: size == 0 Bytes")
		return nil, errors.New("Invalid response body")
	}

	go c.save(params, x, y, z, &bodyBytes)
	go c.memoryMapStore(params, x, y, z, &bodyBytes)

	duration := time.Since(start)
	c.logDebug("Serving " + strconv.Itoa(len(bodyBytes)) + " Bytes to client (took " + duration.String() + ")")

	return bodyBytes, nil
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

func (c *Cache) load(requestParams *url.Values, x string, y string, z string) ([]byte, error) {
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

	c.logDebug("ModTime for " + fp.FullPath + ": " + t.ModTime().String())

	if c.isFileOutdated(t.ModTime()) {
		return nil, errors.New("Tile is too old!")
	}

	duration := time.Since(start)
	c.logDebug("Loaded tile from " + fp.FullPath + " (took " + duration.String() + ")")

	return data, nil
}

func (c *Cache) save(requestParams *url.Values, x string, y string, z string, data *[]byte) error {
	start := time.Now()

	fp := c.makeFilepath(requestParams, x, y, z)

	c.logDebug("Saving " + strconv.Itoa(len(*data)) + " Bytes to " + fp.FullPath)

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
	c.logDebug("Tile with " + strconv.Itoa(len(*data)) + " Bytes successfully saved to " + fp.FullPath + " (took " + duration.String() + ")")
	return nil
}

func (c *Cache) serve(w http.ResponseWriter, req *http.Request) {
	// route format: /{route}/{s}/{z}/{y}/{x}/?params
	start := time.Now()

	c.logDebug("Received request with RequestURI [" + req.RequestURI + "]")

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

	c.logDebug("Params found in route: s=[" + s + "], x=[" + x + "], y=[" + y + "], z=[" + z + "]")

	params := req.URL.Query()
	c.logDebug("Request params found : " + fmt.Sprint(params))

	var data []byte
	var err error

	data, err = c.memoryMapLoad(&params, x, y, z)

	if err != nil {
		c.logDebug("Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from MemoryMap, will try HDD...")
		data, err = c.load(&params, x, y, z)

		if err != nil {
			c.logDebug("Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from HDD, will request it from server...")
		} else {
			c.logDebug("Tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] found in HDD-Storage!")
			c.Stats.BytesServedFromHDD += len(data)
			c.memoryMapStore(&params, x, y, z, &data)
		}
	} else {
		c.logDebug("Tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] found in MemoryMap!")
		c.Stats.BytesServedFromMemory += len(data)
	}

	if err != nil {
		c.logDebug("Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "], reason: " + err.Error())
		c.logDebug("Sending request to server...")

		sourceHeader := req.Header.Clone()

		data, err = c.request(x, y, z, s, &params, &sourceHeader)

		if err != nil {
			c.logWarn("Could not fetch tile for x=[" + x + "], y=[" + y + "], z=[" + z + "].")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
			return
		} else {
			c.logDebug("Fetched tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from server (" + strconv.Itoa(len(data)) + " Bytes)!")
			c.Stats.BytesServedFromOrigin += len(data)
		}
	} else {
		c.logDebug("Loaded tile for x=[" + x + "], y=[" + y + "], z=[" + z + "] from cache (" + strconv.Itoa(len(data)) + " Bytes)!")
		c.Stats.BytesServedFromCache += len(data)
	}

	//c.LogStats()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	duration := time.Since(start)
	c.logDebug("Processing request with RequestURI: [" + req.RequestURI + "] took " + duration.String())

	w.Write(data)
}
