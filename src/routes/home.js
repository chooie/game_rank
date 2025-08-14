import { Router } from "express";

export default function createHomeRoutes({ db }) {
  const router = Router();

  const clicked_route = "/home/htmx/clicked";
  const users_route = "/users";

  const insert_user = db.prepare("INSERT INTO users (name, age) VALUES (?, ?)");
  const get_all_users = db.prepare("SELECT * FROM users");

  router.get("/", (req, res) => {
    res.render("home", {
      title: `Home — ${req.app.locals.APP_NAME}`,
      page: "home",
      message: "Hello from Handlebars!",
      htmx_routes: { clicked: clicked_route, users: users_route },
    });
  });

  router.get(clicked_route, (req, res) => {
    res.render("home__server_time", {
      layout: false,
      message: "Server says hello 👋",
      serverTime: new Date().toLocaleTimeString(),
    });
  });

  router.get(users_route, (req, res) => {
    const users = get_all_users.all();
    res.render("home__users", { layout: false, users });
  });

  router.post(users_route, (req, res) => {
    try {
      insert_user.run("Alice", 25);
      insert_user.run("Bob", 30);
      const users = get_all_users.all();
      res.render("home__users", { layout: false, users });
    } catch (err) {
      console.error(err);
      res.status(500).send("Error inserting users");
    }
  });

  return router;
}
