import dotenv from "dotenv";
import express from "express";
import { engine } from "express-handlebars";
import path from "node:path";
import { fileURLToPath } from "node:url";

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

// Handlebars
app.engine(
  "handlebars",
  engine({
    defaultLayout: "base", // templates/layouts/base.handlebars
    extname: ".handlebars",
    helpers: {
      shout: (text = "") => String(text).toUpperCase(),
    },
  }),
);
app.set("view engine", "handlebars");
app.set("views", VIEWS_DIR);

// App locals (available in all templates)
app.locals.appName = APP_NAME;
app.locals.isDev = IS_DEV;

// Middleware
app.use(express.json());

// Everything can be served from the 'public' directory
app.use(express.static(path.join(__dirname, "../public")));

// Routes
app.get("/", (req, res) => {
  res.render("home", {
    title: `Home — ${APP_NAME}`,
    message: "Hello from Handlebars!",
  });
});

app.get("/healthz", (_req, res) => res.send("ok"));

// 404 (optional)
app.use((req, res) => {
  res.status(404).render("404", { title: `Not Found — ${APP_NAME}` });
});

// Start
app.listen(PORT, () => {
  console.log(
    `🚀 ${APP_NAME} running at http://localhost:${PORT} (${IS_DEV ? "dev" : "prod"})`,
  );
});
