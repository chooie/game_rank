import path from "node:path";
import { fileURLToPath } from "node:url";

import dotenv from "dotenv";
import { Eta } from "eta";
import express from "express";
import { engine } from "express-handlebars";

// Load env
dotenv.config();

// Resolve paths relative to this file
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const VIEWS_DIR = path.join(__dirname, "templates");

const app = express();

const APP_NAME = "Game Rank";
const PORT = Number(process.env.PORT) || 3000;
const IS_DEV = process.env.NODE_ENV === "development";
const IS_DEBUG = true;

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

const eta = new Eta({
  views: VIEWS_DIR,
  cache: true && !IS_DEV,
});

// Middleware
app.use(express.json());

// Everything can be served from the 'public' directory
app.use(express.static(path.join(__dirname, "../public")));

const clicked_route = "/home/htmx/clicked";

// ==========================
// Routes
// ==========================

app.get("/", (req, res, next) => {
  const html = eta.render("home", {
    title: `Home — ${APP_NAME}`,
    page: "home",
    message: "Hello from Eta!",
    htmx_routes: { clicked: clicked_route },
  });
  res.send(html);
});

app.get(clicked_route, (req, res) => {
  res.render("home__server_time", {
    layout: false, // important: no main layout
    message: "Server says hello 👋",
    serverTime: new Date().toLocaleTimeString(),
  });
});

app.get("/healthz", (_req, res) => res.send("ok"));

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
