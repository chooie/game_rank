import { Router } from "express";

export default function createHomeRoutes({ db }) {
  const router = Router();

  const clicked_route = "/home/htmx/clicked";
  const insert_users_route = "/insert-users";

  const insertUser = db.prepare("INSERT INTO users (name, age) VALUES (?, ?)");
  const getAllUsers = db.prepare("SELECT * FROM users");

  router.get("/", (req, res) => {
    res.render("home", {
      title: `Home — ${req.app.locals.APP_NAME}`,
      page: "home",
      message: "Hello from Handlebars!",
      htmx_routes: { clicked: clicked_route, insert_users: insert_users_route },
    });
  });

  router.get(clicked_route, (req, res) => {
    res.render("home__server_time", {
      layout: false,
      message: "Server says hello 👋",
      serverTime: new Date().toLocaleTimeString(),
    });
  });

  router.post(insert_users_route, (req, res) => {
    try {
      insertUser.run("Alice", 25);
      insertUser.run("Bob", 30);
      const users = getAllUsers.all();
      res.render("home__users", { layout: false, users });
    } catch (err) {
      console.error(err);
      res.status(500).send("Error inserting users");
    }
  });

  router.get("/users", (req, res) => {
    const users = getAllUsers.all();
    res.render("users", { layout: false, users });
  });

  return router;
}
