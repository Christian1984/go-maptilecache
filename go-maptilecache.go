package maptilecache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/djherbis/times"
)

type FilePath struct {
	Path     string
	FullPath string
}

type CacheStats struct {
	BytesServedFromCache  int
	BytesServedFromOrigin int
}

type Cache struct {
	Route      []string
	UrlScheme  string
	TimeToLive time.Duration
	ApiKey     string
	Stats      CacheStats
	Logger     LoggerConfig
}

func New(route []string, urlScheme string, TimeToLiveDays time.Duration, apiKey string) (Cache, error) {
	c := Cache{
		Route:      route,
		UrlScheme:  urlScheme,
		TimeToLive: TimeToLiveDays,
		ApiKey:     apiKey,
		Logger:     LoggerConfig{LogPrefix: "Cache[" + strings.Join(route, "/") + "]"},
	}

	if len(route) < 1 {
		return c, errors.New("Could not initialize Cache, reason: Route invalid, must have at least one entry!")
	}

	routeString := strings.Join(route, "/")

	go c.removeOutdatedTiles()

	http.HandleFunc("/"+routeString+"/", c.serve)
	fmt.Println("New Cache initialized on route /" + routeString + "/")

	return c, nil
}

func (c *Cache) isFileOutdated(modtime time.Time) bool {
	age := time.Now().Sub(modtime)
	return age > c.TimeToLive
}

func (c *Cache) removeOutdatedTiles() {
	c.logInfo("Cleaning cache...")

	root := filepath.Join(append([]string{"."}, c.Route...)...)

	if _, statErr := os.Stat(root); statErr != nil {
		c.logInfo("Cache directory not yet initialized. Aborting cleanup!")
		return
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			infoString := fmt.Sprintf("Inspecting file [%s] => size: %d Bytes, modtime: %s", path, info.Size(), info.ModTime().String())
			c.logDebug(infoString)

			if c.isFileOutdated(info.ModTime()) {
				c.logDebug("[" + path + "] is outdated. Removing file from cache...")
				removeErr := os.Remove(path)

				if removeErr != nil {
					c.logWarn("Could not remove [" + path + "]")
					return nil
				}

				c.logDebug("Removed file [" + path + "]")
			} else {
				c.logDebug("File [" + path + "] is current.")
			}
		}

		return nil
	})

	if err != nil {
		c.logWarn("Could not clean cache, reason: " + err.Error())
		return
	}

	c.logInfo("Cache cleaned!")
}

func (c *Cache) request(x string, y string, z string, s string, sourceHost string, sourceHeader *http.Header) ([]byte, error) {
	url := c.UrlScheme
	url = strings.Replace(url, "{s}", s, 1)
	url = strings.Replace(url, "{x}", x, 1)
	url = strings.Replace(url, "{y}", y, 1)
	url = strings.Replace(url, "{z}", z, 1)

	c.logDebug("Requesting tile from " + url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logError("Could not create request, reason: " + err.Error())
		return nil, err
	}

	req.Header = *sourceHeader

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

	c.logInfo("Received " + strconv.Itoa(len(bodyBytes)) + " Bytes from " + url)

	if len(bodyBytes) == 0 {
		c.logError("Invalid response body, reason: size == 0 Bytes")
		return nil, errors.New("Invalid response body")
	}

	go c.save(x, y, z, &bodyBytes)

	return bodyBytes, nil
}

func (c *Cache) makeFilepath(x string, y string, z string) FilePath {
	pathArray := append([]string{"."}, c.Route...)
	pathArray = append(pathArray, z, y)
	path := filepath.Join(pathArray...)
	fullPath := filepath.Join(path, x+".png")

	return FilePath{
		Path:     path,
		FullPath: fullPath,
	}
}

func (c *Cache) load(x string, y string, z string) ([]byte, error) {
	fp := c.makeFilepath(x, y, z)
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

	return data, nil
}

func (c *Cache) save(x string, y string, z string, data *[]byte) error {
	fp := c.makeFilepath(x, y, z)

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

	c.logDebug("Tile successfully saved to " + fp.FullPath)
	return nil
}

func (c *Cache) serve(w http.ResponseWriter, req *http.Request) {
	// route format: /{route}/{s}/{z}/{y}/{x}/
	c.logDebug("Received request with RequestURI: [" + req.RequestURI + "]")

	requestUri := strings.Split(req.RequestURI, "/")

	if len(requestUri) < 5+len(c.Route) {
		c.logError("Bad Request: Not enough arguments in route [" + req.RequestURI + "]")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return
	}

	s := requestUri[1+len(c.Route)]
	z := requestUri[2+len(c.Route)]
	y := requestUri[3+len(c.Route)]
	x := requestUri[4+len(c.Route)]

	c.logDebug("Params found: s=[" + s + "], x=[" + x + "], y=[" + y + "], z=[" + z + "]")

	data, err := c.load(x, y, z)

	if err != nil {
		c.logDebug("Could not load tile for x=[" + x + "], y=[" + y + "], z=[" + z + "], reason: " + err.Error())
		c.logDebug("Sending request to server...")

		sourceHost := req.Host
		sourceHeader := req.Header.Clone()

		data, err = c.request(x, y, z, s, sourceHost, &sourceHeader)

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

	c.LogStats()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))

	w.Write(data)
}
