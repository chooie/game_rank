package main

import (
	"bytes"
	"context"
	"database/sql"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB // global DB handle

func main() {
	// ── DB setup ────────────────────────────────────────────────────────────────
	var err error
	db, err = openDB("foobar.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := initSchema(ctx, db); err != nil {
		log.Fatal(err)
	}

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
	RouteHTMXUsers:   HTMXUsersHandler,
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

var sampleNames = []string{
	"Alice", "Bob", "Charlie", "Diana",
	"Eve", "Frank", "Grace", "Heidi",
	"Ivan", "Judy", "Mallory", "Niaj",
}

type UsersParams struct {
	Users []map[string]any
}

// HTMXUsersHandler returns the users partial on both GET (list) and POST (insert + list)
func HTMXUsersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := fetchUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		RenderPartial("src2/templates/home__users.tmpl", UsersParams{Users: users}, w)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name := sampleNames[rand.Intn(len(sampleNames))]

		age := rand.Intn(60) + 18 // random age between 18 and 77

		if _, err := db.ExecContext(r.Context(),
			`INSERT INTO users(name, age) VALUES(?, ?)`, name, age); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// After insert, return the updated list partial (good for HTMX)
		users, err := fetchUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		RenderPartial("src2/templates/home__users.tmpl", UsersParams{Users: users}, w)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func fetchUsers(ctx context.Context) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, name, age FROM users ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]any
	for rows.Next() {
		var id int64
		var name sql.NullString
		var age sql.NullInt64
		if err := rows.Scan(&id, &name, &age); err != nil {
			return nil, err
		}
		users = append(users, map[string]any{
			"id":   id,
			"name": name.String,
			"age":  age.Int64,
		})
	}
	return users, rows.Err()
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

// ─────────────────────────────────────────────────────────────────────────────
// DB
// ─────────────────────────────────────────────────────────────────────────────

func openDB(filename string) (*sql.DB, error) {
	// DSN: enable WAL and a saner tx lock for web apps
	dsn := "file:" + filename + "?_pragma=journal_mode(WAL)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite + database/sql: safest is single writer connection
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	return db, nil
}

func initSchema(ctx context.Context, db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	age INTEGER
);

CREATE TABLE IF NOT EXISTS games (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	rank INTEGER NOT NULL
);
`
	_, err := db.ExecContext(ctx, ddl)
	return err
}
