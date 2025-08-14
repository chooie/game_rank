import path from "node:path";
import { fileURLToPath } from "node:url";

import Database from "better-sqlite3";
import dotenv from "dotenv";
import express from "express";
import { engine } from "express-handlebars";

// Load env
dotenv.config();

// Resolve paths relative to this file
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const VIEWS_DIR = path.join(__dirname, "templates");

const db = new Database("foobar.db");
db.pragma("journal_mode = WAL");

db.exec(`
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    age INTEGER
  )
`);

const app = express();

const APP_NAME = "Game Rank";
const PORT = Number(process.env.PORT) || 3000;
const IS_DEV = process.env.NODE_ENV === "development";

// Handlebars
app.engine(
  "hbs",
  engine({
    defaultLayout: "base", // templates/layouts/base.hbs
    extname: ".hbs",
    helpers: {
      debug_json: (context) => JSON.stringify(context, null, 2),
      shout: (text = "") => String(text).toUpperCase(),
    },
  }),
);
app.set("view engine", "hbs");
app.set("views", VIEWS_DIR);

// Middleware
app.use(express.json());

// Everything can be served from the 'public' directory
app.use(express.static(path.join(__dirname, "../public")));

const clicked_route = "/home/htmx/clicked";
const insert_users_route = "/insert-users";

// ==========================
// Routes
// ==========================

app.get("/", (req, res) => {
  res.render("home", {
    title: `Home — ${APP_NAME}`,
    page: "home",
    message: "Hello from Handlebars!",
    htmx_routes: { clicked: clicked_route, insert_users: insert_users_route },
  });
});

app.get(clicked_route, (req, res) => {
  res.render("home__server_time", {
    layout: false, // important: no main layout
    message: "Server says hello 👋",
    serverTime: new Date().toLocaleTimeString(),
  });
});

app.get("/healthz", (_req, res) => res.send("ok"));

const insertUser = db.prepare("INSERT INTO users (name, age) VALUES (?, ?)");
const getAllUsers = db.prepare("SELECT * FROM users");

app.post(insert_users_route, (req, res) => {
  try {
    insertUser.run("Alice", 25);
    insertUser.run("Bob", 30);

    const users = getAllUsers.all(); // Returns an array of objects
    res.render("home__users", { layout: false, users }); // Pass array to template
  } catch (err) {
    console.error(err);
    res.status(500).send("Error inserting users");
  }
});

app.get("/users", (req, res) => {
  const users = getAllUsers.all(); // Returns an array of objects
  res.render("users", { layout: false, users }); // Pass array to template
});

// 404 (optional)
app.use((req, res) => {
  res
    .status(404)
    .render("404", { title: `Not Found — ${APP_NAME}`, page: "404" });
});

// Start
app.listen(PORT, () => {
  console.log(
    `🚀 ${APP_NAME} running at http://localhost:${PORT} (${IS_DEV ? "dev" : "prod"})`,
  );
});
