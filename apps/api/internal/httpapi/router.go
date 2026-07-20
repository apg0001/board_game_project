package httpapi

import (
	"log/slog"
	"net/http"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/room"
)

func NewRouter(cfg config.Config, logger *slog.Logger, games catalog.Catalog, guests *guest.Service, rooms *room.Service, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()
	api := Handler{games: games, guests: guests, rooms: rooms, hub: hub}

	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/guests", api.createGuest)
	mux.HandleFunc("GET /api/me", api.me)
	mux.HandleFunc("POST /api/rooms", api.createRoom)
	mux.HandleFunc("POST /api/rooms/join", api.joinRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", api.getRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/ready", api.setReady)
	mux.HandleFunc("GET /api/games", api.listGames)
	mux.HandleFunc("GET /api/games/recommend", api.recommendGames)
	mux.HandleFunc("GET /ws", api.websocket)

	return withCORS(cfg, withRequestLog(logger, mux))
}
