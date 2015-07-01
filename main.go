package main

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
)

func hello(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %s!", c.URLParams["name"])
}

func serveIndex(c web.C, w http.ResponseWriter, r *http.Request) {
	indexTemplate.Execute(w, nil)
}

var indexTemplate = template.Must(template.ParseFiles("templates/index.html"))

func main() {
	goji.Get("/", serveIndex)
	goji.Get("/hello/:name", hello)
	goji.Serve()
}
