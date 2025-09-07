package main

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

func main() {
	mux := http.NewServeMux()

	// Static files at /public/*
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/public/", http.StripPrefix("/public/", fs))

	// Register typed routes
	RegisterRoutes(mux, routes)

	log.Println("Starting server on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}

// Route type
type Route string

// Route constants
const (
	RouteHome        Route = "/"
	RouteAnimals     Route = "/animals"
	RouteHTMXClicked Route = "/home/htmx/clicked"
	RouteHTMXUsers   Route = "/home/htmx/users"
)

var routes = map[Route]http.HandlerFunc{
	RouteHome:        HomeHandler,
	RouteAnimals:     AnimalsHandler,
	RouteHTMXClicked: HTMXClickedHandler,
	// RouteHTMXUsers:   htmxUsersHandler,
}

func RegisterRoutes(mux *http.ServeMux, routes map[Route]http.HandlerFunc) {
	for path, handler := range routes {
		mux.HandleFunc(string(path), handler)
	}
}

type BaseTemplateParams struct {
	Base
}

type Base struct {
	Title       string
	Page        string
	Description string
}

type HomeParams struct {
	Base
	Message    string
	HTMXRoutes HomeHTMXRoutes
}

type HomeHTMXRoutes struct {
	Clicked string
	Users   string
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		NotFound(w, r)
		return
	}

	p := HomeParams{
		Base: Base{
			Title:       "Home Page Title",
			Page:        "home",
			Description: "Hello, from Charlie!",
		},
		HTMXRoutes: HomeHTMXRoutes{
			Clicked: string(RouteHTMXClicked),
			Users:   string(RouteHTMXUsers),
		},
	}

	RenderTemplate("src2/templates/home.tmpl", p, w)
}

type AnimalsParams struct {
	Base
	Message string
}

func AnimalsHandler(w http.ResponseWriter, r *http.Request) {
	p := BaseTemplateParams{
		Base: Base{
			Title:       "Animals Page Title",
			Page:        "animals",
			Description: "A page about animals.",
		},
	}

	RenderTemplate("src2/templates/animals.tmpl", p, w)
}

type ClickedParams struct {
	Message    string
	ServerTime string
}

func HTMXClickedHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	p := ClickedParams{
		Message:    "You just clicked!",
		ServerTime: now.Format("2006-01-02 15:04:05"), // YYYY-MM-DD HH:mm:ss
	}

	RenderPartial("src2/templates/clicked.tmpl", p, w)
}

func NotFound(w http.ResponseWriter, r *http.Request) {
	p := BaseTemplateParams{
		Base: Base{
			Title:       "404 Not Found",
			Page:        "404",
			Description: "Sorry, we couldn’t find that page.",
		},
	}
	RenderTemplate("src2/templates/404.tmpl", p, w)
}

func RenderTemplate(template_path string, template_params any, w http.ResponseWriter) {
	tmpl := template.Must(
		template.New("layout").
			Option("missingkey=error").
			ParseFiles("src2/templates/layout.tmpl", template_path),
	)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", template_params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Only write to the response if template execution succeeded
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}

func RenderPartial(templatePath string, templateParams any, w http.ResponseWriter) {
	// Name the root template the same as the file's base name.
	root := filepath.Base(templatePath)

	tmpl := template.Must(
		template.
			New(root).
			Option("missingkey=error").
			ParseFiles(templatePath),
	)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateParams); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	buf.WriteTo(w)
}
