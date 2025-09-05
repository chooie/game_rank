package main

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed templates/*.gohtml
var tmplFS embed.FS

var homeTmpl = template.Must(
	template.New("all").ParseFS(tmplFS, "templates/layout.gohtml", "templates/home.gohtml"),
)

type HomeParams struct {
	Title      string
	Page       string
	Message    string
	HTMXRoutes struct {
		Clicked string
		Users   string
	}
}

func renderLayout(w http.ResponseWriter, status int, p HomeParams) error {
	var buf bytes.Buffer
	if err := homeTmpl.ExecuteTemplate(&buf, "layout", p); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, _ = buf.WriteTo(w)
	return nil
}

func Home(w http.ResponseWriter, p HomeParams) error {
	// Render the base layout; it will pull in the "content" block from home.gohtml
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return renderLayout(w, http.StatusOK, p)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	p := HomeParams{
		Title:   "404 Not Found",
		Page:    "404",
		Message: "Sorry, we couldn’t find that page.",
	}
	// Render with a 404 status
	if err := renderLayout(w, http.StatusNotFound, p); err != nil {
		log.Println("404 template error:", err)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func main() {
	// Serve static files from ./public at /public/*
	// e.g. /public/dist/css/home.css -> ./public/dist/css/home.css
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/public/", http.StripPrefix("/public/", fs))

	// Home route (only exact "/"); everything else -> 404
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			NotFound(w, r)
			return
		}

		var p HomeParams
		p.Title = "Home Page Title"
		p.Page = "home"
		p.Message = "Hello, from Charlie!"
		p.HTMXRoutes.Clicked = "/clicked"
		p.HTMXRoutes.Users = "/users"

		if err := Home(w, p); err != nil {
			log.Println("template error:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	log.Println("Starting server on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
