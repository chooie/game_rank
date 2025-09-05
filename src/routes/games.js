import { Router } from "express";

const games_route = "/games";
const games_reorder_route = "/games/reorder";

export default function createGameRoutes({ db }) {
  const router = Router();

  // Queries
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
  const update_rank = db.prepare("UPDATE games SET rank = ? WHERE id = ?");
  const select_ids = db.prepare("SELECT id FROM games");

  // Insert transaction (shift others if inserting in the middle)
  const insertAtRankTx = db.transaction((title, rank) => {
    shift_ranks.run(rank);
    insert_game.run(title, rank);
  });

  // Reorder transaction: ids[] is the new DOM order (top → bottom)
  const reorderTx = db.transaction((orderedIds) => {
    for (let i = 0; i < orderedIds.length; i++) {
      // rank is 1-based
      update_rank.run(i + 1, orderedIds[i]);
    }
  });

  // Helpers
  function renderFull(res, status, data) {
    return res.status(status).render("games", { title: "Games", ...data });
  }
  function renderPartial(res, status, data) {
    // This should render the same HTML that replaces #games-wrapper
    return res.status(status).render("games__list", { layout: false, ...data });
  }
  function routes() {
    return { games: games_route, games_reorder: games_reorder_route };
  }

  router.get(games_route, (req, res) => {
    const games = get_all_games.all();
    return renderFull(res, 200, { page: "games", games, routes: routes() });
  });

  router.post(games_route, (req, res) => {
    const isHtmx = !!req.get("HX-Request");

    try {
      const { title, rank: rankRaw } = req.body ?? {};
      const errors = {};
      const old = { title: (title ?? "").trim(), rank: rankRaw };

      if (!old.title) errors.title = "Title is required.";
      if (old.title && old.title.length < 3) {
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
        const payload = { games, routes: routes(), errors, old };
        return isHtmx
          ? renderPartial(res, 200, payload)
          : renderFull(res, 422, payload);
      }

      if (rank <= maxRank) insertAtRankTx(old.title, rank);
      else insert_game.run(old.title, rank);

      const games = get_all_games.all();
      return isHtmx
        ? renderPartial(res, 200, { games, routes: routes() })
        : res.redirect(games_route);
    } catch (err) {
      console.error(err);
      const games = get_all_games.all();
      const errors = { _form: "Failed to add game. Please try again." };
      const old = {
        title: (req.body?.title ?? "").trim(),
        rank: req.body?.rank,
      };
      const payload = { games, routes: routes(), errors, old };
      return req.get("HX-Request")
        ? renderPartial(res, 500, payload)
        : renderFull(res, 500, payload);
    }
  });

  // NEW: reorder endpoint used by hx-post on drag end
  router.post(games_reorder_route, (req, res) => {
    const isHtmx = !!req.get("HX-Request");

    // `game` comes from <input name="game" value="{{id}}"> in DOM order.
    // It can be a string (single) or an array (multiple).
    let idsRaw = req.body?.game;
    if (idsRaw == null) {
      const games = get_all_games.all();
      const errors = { _form: "No items submitted." };
      const payload = { games, routes: routes(), errors };
      return isHtmx
        ? renderPartial(res, 400, payload)
        : renderFull(res, 400, payload);
    }
    if (!Array.isArray(idsRaw)) idsRaw = [idsRaw];

    // Parse & validate
    const orderedIds = idsRaw
      .map((v) => Number.parseInt(String(v), 10))
      .filter((n) => Number.isFinite(n));

    if (orderedIds.length !== idsRaw.length) {
      const games = get_all_games.all();
      const errors = { _form: "Invalid item identifiers." };
      const payload = { games, routes: routes(), errors };
      return isHtmx
        ? renderPartial(res, 400, payload)
        : renderFull(res, 400, payload);
    }

    // Optional safety: ensure submitted ids match exactly the set in DB
    const dbIds = select_ids.all().map((r) => r.id);
    const sameCardinality = dbIds.length === orderedIds.length;
    const sameSet =
      sameCardinality &&
      new Set(dbIds).size === new Set(orderedIds).size &&
      orderedIds.every((id) => dbIds.includes(id));

    if (!sameSet) {
      const games = get_all_games.all();
      const errors = { _form: "Submitted list does not match current items." };
      const payload = { games, routes: routes(), errors };
      return isHtmx
        ? renderPartial(res, 409, payload)
        : renderFull(res, 409, payload);
    }

    // Apply ranks (1-based) in a single transaction
    try {
      reorderTx(orderedIds);
    } catch (e) {
      console.error(e);
      const games = get_all_games.all();
      const errors = { _form: "Failed to reorder. Please try again." };
      const payload = { games, routes: routes(), errors };
      return isHtmx
        ? renderPartial(res, 500, payload)
        : renderFull(res, 500, payload);
    }

    // Return updated list (HTMX swap) or full page
    const games = get_all_games.all();
    return isHtmx
      ? renderPartial(res, 200, { games, routes: routes() })
      : renderFull(res, 200, { page: "games", games, routes: routes() });
  });

  return router;
}
