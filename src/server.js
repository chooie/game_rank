import path from "node:path";
import { fileURLToPath } from "node:url";

import Database from "better-sqlite3";
import dotenv from "dotenv";
import express from "express";
import { engine } from "express-handlebars";

import registerRoutes from "./routes/_routes.js";

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

db.exec(`
  CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    rank INTEGER NOT NULL
  )
`);

const app = express();

const APP_NAME = "Game Rank";
app.locals.APP_NAME = APP_NAME; // available in all res.render() calls

const PORT = Number(process.env.PORT) || 3000;
const IS_DEV = process.env.NODE_ENV === "development";

// Handlebars
app.engine(
  "hbs",
  engine({
    defaultLayout: "base", // templates/layouts/base.hbs
    extname: ".hbs",
    layoutsDir: path.join(VIEWS_DIR, "layouts"),
    // Make *all* .hbs files under templates/ available as partials,
    // so sibling partials like templates/games__list.hbs work:
    partialsDir: VIEWS_DIR,
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
// Needed for form submissions
app.use(express.urlencoded({ extended: false }));

// Everything can be served from the 'public' directory
app.use(express.static(path.join(__dirname, "../public")));

registerRoutes(app, db);

// Start
app.listen(PORT, () => {
  console.log(
    `🚀 ${APP_NAME} running at http://localhost:${PORT} (${IS_DEV ? "dev" : "prod"})`,
  );
});
