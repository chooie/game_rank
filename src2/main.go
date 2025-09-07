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
	"strconv"
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
	RouteHome         Route = "/"
	RouteAnimals      Route = "/animals"
	RouteHTMXClicked  Route = "/home/htmx/clicked"
	RouteHTMXUsers    Route = "/home/htmx/users"
	RouteGames        Route = "/games"
	RouteGamesReorder Route = "/games/reorder"
)

var routes = map[Route]http.HandlerFunc{
	RouteHome:         HomeHandler,
	RouteAnimals:      AnimalsHandler,
	RouteHTMXClicked:  HTMXClickedHandler,
	RouteHTMXUsers:    HTMXUsersHandler,
	RouteGames:        GamesHandler,
	RouteGamesReorder: GamesReorderHandler,
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

	w.WriteHeader(http.StatusOK)
	RenderTemplate(w, p, []string{
		"src2/templates/layout.tmpl",
		"src2/templates/home.tmpl",
	})
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

	w.WriteHeader(http.StatusOK)
	RenderTemplate(w, p, []string{
		"src2/templates/layout.tmpl",
		"src2/templates/animals.tmpl",
	})
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

	w.WriteHeader(http.StatusOK)
	// This partial is assumed to be raw (no {{define}}). If you wrap it in define,
	// switch to RenderPartialNamed and pass the defined name.
	RenderPartial("src2/templates/clicked.tmpl", p, w)
}

var sampleNames = []string{
	"Alice", "Bob", "Charlie", "Diana",
	"Eve", "Frank", "Grace", "Heidi",
	"Ivan", "Judy", "Mallory", "Niaj",
}

type UsersParams struct {
	Users          []map[string]any
	RouteHTMXUsers string
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

		p := UsersParams{
			Users:          users,
			RouteHTMXUsers: string(RouteHTMXUsers),
		}
		w.WriteHeader(http.StatusOK)
		RenderPartial("src2/templates/home__users.tmpl", p, w)

	case http.MethodPost:
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

		p := UsersParams{
			Users:          users,
			RouteHTMXUsers: string(RouteHTMXUsers),
		}
		w.WriteHeader(http.StatusOK)
		RenderPartial("src2/templates/home__users.tmpl", p, w)

	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		if _, err := db.ExecContext(r.Context(),
			`DELETE FROM users WHERE id = ?`, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return the updated users list (HTMX will swap it in)
		users, err := fetchUsers(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		p := UsersParams{
			Users:          users,
			RouteHTMXUsers: string(RouteHTMXUsers),
		}
		w.WriteHeader(http.StatusOK)
		RenderPartial("src2/templates/home__users.tmpl", p, w)

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

// ─────────────────────────────────────────────────────────────────────────────
// Games
// ─────────────────────────────────────────────────────────────────────────────

type Game struct {
	ID    int64
	Title string
	Rank  int
}

type GamesRoutes struct {
	Games        string
	GamesReorder string
}

type GamesPageParams struct {
	Base
	Routes GamesRoutes
	Old    *struct {
		Title string
		Rank  int
	}
	Errors map[string]string
	Games  []Game
}

// GET /games → full page (layout) including wrapper partial
// POST /games → validate/insert, return only wrapper partial for HTMX swap
func GamesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		games, err := fetchGames(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p := GamesPageParams{
			Base: Base{
				Title:       "Games",
				Page:        "games",
				Description: "Rank your games.",
			},
			Routes: GamesRoutes{
				Games:        string(RouteGames),
				GamesReorder: string(RouteGamesReorder),
			},
			Games:  games,
			Errors: map[string]string{},
		}
		// Include the partial file that defines {{ define "games__list" }}
		w.WriteHeader(http.StatusOK)
		RenderTemplate(w, p, []string{
			"src2/templates/layout.tmpl",
			"src2/templates/games.tmpl",
			"src2/templates/games__list.tmpl",
		})

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		rankStr := r.FormValue("rank")

		var rank int
		if rankStr != "" {
			if n, err := strconv.Atoi(rankStr); err == nil {
				rank = n
			}
		}

		errors := map[string]string{}
		if title == "" {
			errors["title"] = "Title is required"
		}
		if rank <= 0 {
			// Default to next rank if not provided
			if next, err := nextRank(r.Context()); err == nil {
				if rank == 0 {
					rank = next
				} else {
					errors["rank"] = "Rank must be ≥ 1"
				}
			}
		}

		if len(errors) == 0 {
			if _, err := db.ExecContext(r.Context(),
				`INSERT INTO games(title, rank) VALUES(?, ?)`, title, rank); err != nil {
				errors["_form"] = "Could not save game"
				log.Println("insert game:", err)
			}
		}

		games, err := fetchGames(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		p := GamesPageParams{
			Routes: GamesRoutes{
				Games:        string(RouteGames),
				GamesReorder: string(RouteGamesReorder),
			},
			Old: &struct {
				Title string
				Rank  int
			}{Title: title, Rank: rank},
			Errors: errors,
			Games:  games,
		}
		w.WriteHeader(http.StatusOK)
		// Return ONLY the named partial (file uses {{ define "games__list" }})
		RenderPartialNamed("src2/templates/games__list.tmpl", "games__list", p, w)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /games/reorder → receives DOM order of inputs named "game" with values = IDs
func GamesReorderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ids := r.Form["game"] // DOM order
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, idStr := range ids {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			_ = tx.Rollback()
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE games SET rank = ? WHERE id = ?`, i+1, id); err != nil {
			_ = tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	games, err := fetchGames(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := GamesPageParams{
		Routes: GamesRoutes{
			Games:        string(RouteGames),
			GamesReorder: string(RouteGamesReorder),
		},
		Errors: map[string]string{},
		Games:  games,
	}

	w.WriteHeader(http.StatusOK)
	// Return the same named partial after reorder
	RenderPartialNamed("src2/templates/games__list.tmpl", "games__list", p, w)
}

func fetchGames(ctx context.Context) ([]Game, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, title, rank FROM games ORDER BY rank ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Game
	for rows.Next() {
		var g Game
		if err := rows.Scan(&g.ID, &g.Title, &g.Rank); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func nextRank(ctx context.Context) (int, error) {
	var maxRank sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT MAX(rank) FROM games`).Scan(&maxRank)
	if err != nil {
		return 1, err
	}
	if !maxRank.Valid {
		return 1, nil
	}
	return int(maxRank.Int64) + 1, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Rendering
// ─────────────────────────────────────────────────────────────────────────────

func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound) // send 404

	p := BaseTemplateParams{
		Base: Base{
			Title:       "404 Not Found",
			Page:        "404",
			Description: "Sorry, we couldn’t find that page.",
		},
	}

	RenderTemplate(w, p, []string{
		"src2/templates/layout.tmpl",
		"src2/templates/404.tmpl",
	})
}

// RenderTemplate parses and executes templates from a slice of file paths.
// The first file in the slice should define the "layout" template.
func RenderTemplate(w http.ResponseWriter, templateParams any, templatePaths []string) {
	if len(templatePaths) == 0 {
		http.Error(w, "no templates provided", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(
		template.New("all").
			Option("missingkey=error").
			ParseFiles(templatePaths...),
	)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", templateParams); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Caller must set the status code. We just write the body.
	_, _ = buf.WriteTo(w)
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
	if err := tmpl.Execute(&buf, templateParams); err != nil { // executes root (raw partials)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Helpful default for HTMX; status must be set by caller.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

// RenderPartialNamed parses a file that contains {{ define "name" }} ... {{ end }}
// and executes that named template.
func RenderPartialNamed(templatePath, templateName string, templateParams any, w http.ResponseWriter) {
	tmpl := template.Must(
		template.
			New("partial").
			Option("missingkey=error").
			ParseFiles(templatePath),
	)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, templateName, templateParams); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
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
