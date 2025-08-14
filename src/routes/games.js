import { Router } from "express";

const games_route = "/games";

export default function createGameRoutes({ db }) {
  const router = Router();

  const get_all_games = db.prepare("SELECT * FROM games ORDER BY rank ASC");
  const get_max_rank = db.prepare(
    "SELECT COALESCE(MAX(rank), 0) AS maxRank FROM games",
  );
  const shift_ranks = db.prepare(
    "UPDATE games SET rank = rank + 1 WHERE rank >= ?",
  );
  const insert_game = db.prepare(
    "INSERT INTO games (title, rank) VALUES (?, ?)",
  );

  const insertAtRankTx = db.transaction((title, rank) => {
    shift_ranks.run(rank);
    insert_game.run(title, rank);
  });

  router.get(games_route, (req, res) => {
    const games = get_all_games.all();
    res.render("games", {
      title: "Games",
      page: "games",
      games,
      routes: { games: games_route },
    });
  });

  router.post(games_route, (req, res) => {
    const isHtmx = !!req.get("HX-Request");

    // helpers to render error/success responses
    const renderPartial = (status, data) =>
      res.status(status).render("games__list", { layout: false, ...data });

    const renderFull = (status, data) =>
      res.status(status).render("games", { title: "Games", ...data });

    try {
      const { title, rank: rankRaw } = req.body ?? {};
      const errors = {};
      const old = { title: (title ?? "").trim(), rank: rankRaw };

      // basic validation
      if (!old.title) errors.title = "Title is required.";
      if (old.title.length < 3) {
        errors.title = "The title must be at least 3 characters long.";
      }

      const maxRank = get_max_rank.get().maxRank; // 0 if empty
      const rank = Number.parseInt(rankRaw, 10);
      if (!Number.isFinite(rank)) {
        errors.rank = "Rank must be a number.";
      } else if (rank < 1 || rank > maxRank + 1) {
        errors.rank = `Rank must be between 1 and ${maxRank + 1}.`;
      }

      if (Object.keys(errors).length > 0) {
        const games = get_all_games.all();
        const payload = { games, routes: { games: games_route }, errors, old };
        // ✅ 200 for HTMX so the swap happens; keep 422 for non-HTMX if you like
        return isHtmx ? renderPartial(200, payload) : renderFull(422, payload);
      }

      // insert (auto-shift in the middle, append at end)
      if (rank <= maxRank) {
        insertAtRankTx(old.title, rank);
      } else {
        insert_game.run(old.title, rank);
      }

      const games = get_all_games.all();

      if (isHtmx) {
        // return fresh wrapper (form + updated list)
        return renderPartial(200, { games, routes: { games: games_route } });
      }

      // PRG for non-HTMX success
      return res.redirect(games_route);
    } catch (err) {
      const msg = String(err?.message || "");
      if (msg.includes("UNIQUE constraint failed: games.best_to_worst_rank")) {
        const games = get_all_games.all();
        const errors = { rank: "That rank is already taken. Try another." };
        const old = {
          title: (req.body?.title ?? "").trim(),
          rank: req.body?.rank,
        };
        const payload = { games, routes: { games: games_route }, errors, old };
        return req.get("HX-Request")
          ? res.status(409).render("games__list", { layout: false, ...payload })
          : res.status(409).render("games", { title: "Games", ...payload });
      }

      console.error(err);
      // generic error
      const games = get_all_games.all();
      const errors = { _form: "Failed to add game. Please try again." };
      const old = {
        title: (req.body?.title ?? "").trim(),
        rank: req.body?.rank,
      };
      const payload = { games, routes: { games: games_route }, errors, old };
      return req.get("HX-Request")
        ? res.status(500).render("games__list", { layout: false, ...payload })
        : res.status(500).render("games", { title: "Games", ...payload });
    }
  });

  return router;
}
