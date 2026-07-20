package httpapi

import (
	"log/slog"
	"net/http"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/realtime"
)

func NewRouter(cfg config.Config, logger *slog.Logger, games catalog.Catalog, guests *guest.Service, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()
	api := Handler{games: games, guests: guests, hub: hub}

	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/guests", api.createGuest)
	mux.HandleFunc("GET /api/me", api.me)
	mux.HandleFunc("GET /api/games", api.listGames)
	mux.HandleFunc("GET /api/games/recommend", api.recommendGames)
	mux.HandleFunc("GET /ws", api.websocket)

	return withCORS(cfg, withRequestLog(logger, mux))
}
