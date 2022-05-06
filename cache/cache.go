package cache

import (
	"fmt"
	"net/http"
	"strings"
)

type Cache struct {
	Name           string
	UrlScheme      string
	TimeToLiveDays int
}

func New(name string, urlScheme string, TimeToLiveDays int) Cache {
	c := Cache{
		Name:           name,
		UrlScheme:      urlScheme,
		TimeToLiveDays: TimeToLiveDays,
	}

	http.HandleFunc("/osm/", c.Serve)

	return c
}

/*
func (c *Cache) request(x string, y string, z string, s string) {

}

func (c *Cache) load(x string, y string, z string, s string) {

}

func (c *Cache) save(x string, y string, z string, s string) {

}
*/

func (c *Cache) Serve(w http.ResponseWriter, req *http.Request) {

	fmt.Println("req.RequestURI" + req.RequestURI)

	requestUri := strings.Split(req.RequestURI, "/")

	if len(requestUri) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return
	}

	s := requestUri[2]
	x := requestUri[3]
	y := requestUri[4]
	z := requestUri[5]

	fmt.Printf("Found: s=[%s], x=[%s], y=[%s], z=[%s]\n", s, x, y, z)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Type", "application/json")

	w.Write([]byte("Hello Cache!"))
}
