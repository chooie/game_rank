package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed templates/*.gohtml
var tmplFS embed.FS

var tmpl = template.Must(template.ParseFS(tmplFS, "templates/index.gohtml"))

func handler(w http.ResponseWriter, r *http.Request) {
	data := struct{ Title string }{Title: "Hello, World!"}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Starting server on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
