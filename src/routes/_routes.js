import createHomeRoutes from "./home.js";
import createGameRoutes from "./games.js";

export default function registerRoutes(app, db) {
  app.use("/", createHomeRoutes({ db }));
  app.use("/", createGameRoutes({ db }));

  app.get("/healthz", (_req, res) => res.send("ok"));

  app.use((req, res) => {
    res.status(404).render("404", {
      title: `Not Found — ${req.app.locals.APP_NAME}`,
      page: "404",
    });
  });
}
