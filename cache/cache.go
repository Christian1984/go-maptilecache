package cache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Cache struct {
	Route      []string
	UrlScheme  string
	TimeToLive time.Duration
	ApiKey     string
}

func New(route []string, urlScheme string, TimeToLiveDays time.Duration, apiKey string) (Cache, error) {
	c := Cache{
		Route:      route,
		UrlScheme:  urlScheme,
		TimeToLive: TimeToLiveDays,
		ApiKey:     apiKey,
	}

	if len(route) < 1 {
		return c, errors.New("Could not initialize Cache, reason: Route invalid, must have at least one entry!")
	}

	routeString := strings.Join(route, "/")

	http.HandleFunc("/"+routeString+"/", c.serve)
	fmt.Println("New Cache initialized on route /" + routeString + "/")

	return c, nil
}

func log(message string) {
	fmt.Println(message)
}

func (c *Cache) request(x string, y string, z string, s string) {
	url := c.UrlScheme
	url = strings.Replace(url, "{s}", s, 1)
	url = strings.Replace(url, "{x}", x, 1)
	url = strings.Replace(url, "{y}", y, 1)
	url = strings.Replace(url, "{z}", z, 1)

	log("Requesting tile from " + url)

	resp, err := http.Get(url)

	if err != nil {
		log("Could not request tile, reason: " + err.Error())
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log("Could not request tile, bad status code: " + strconv.Itoa(resp.StatusCode))
		return
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log("Could parse response body, reason: " + err.Error())
		return
	}

	log("Received " + strconv.Itoa(len(bodyBytes)) + " from " + url)
}

/*
func (c *Cache) load(x string, y string, z string, s string) {

}

func (c *Cache) save(x string, y string, z string, s string) {

}
*/

func (c *Cache) serve(w http.ResponseWriter, req *http.Request) {
	// route format: /{route}/{s}/{z}/{y}/{x}/
	log("Received request with RequestURI: [" + req.RequestURI + "]")

	requestUri := strings.Split(req.RequestURI, "/")

	if len(requestUri) < 5+len(c.Route) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return
	}

	s := requestUri[1+len(c.Route)]
	z := requestUri[2+len(c.Route)]
	y := requestUri[3+len(c.Route)]
	x := requestUri[4+len(c.Route)]

	log("Params found: s=[" + s + "], x=[" + s + "], y=[" + s + "], z=[" + s + "]")

	c.request(x, y, z, s)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("Hello Cache!"))
}
