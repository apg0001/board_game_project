package httpapi

import (
	"log/slog"
	"net/http"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/realtime"
)

func NewRouter(cfg config.Config, logger *slog.Logger, games catalog.Catalog, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()
	api := Handler{games: games, hub: hub}

	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("GET /api/games", api.listGames)
	mux.HandleFunc("GET /api/games/recommend", api.recommendGames)
	mux.HandleFunc("GET /ws", api.websocket)

	return withCORS(cfg, withRequestLog(logger, mux))
}
