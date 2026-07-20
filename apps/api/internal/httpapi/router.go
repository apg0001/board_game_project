package httpapi

import (
	"log/slog"
	"net/http"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/chat"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/connection"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/match"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/record"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"
)

func NewRouter(cfg config.Config, logger *slog.Logger, games catalog.Catalog, guests *guest.Service, rooms *room.Service, sessions *session.Service, chats *chat.Service, presence *connection.Service, records *record.Service, matches *match.Service, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()
	api := Handler{games: games, guests: guests, rooms: rooms, sessions: sessions, chats: chats, presence: presence, records: records, matches: matches, hub: hub}

	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/guests", api.createGuest)
	mux.HandleFunc("GET /api/me", api.me)
	mux.HandleFunc("POST /api/rooms", api.createRoom)
	mux.HandleFunc("POST /api/rooms/join", api.joinRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", api.getRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/ready", api.setReady)
	mux.HandleFunc("POST /api/rooms/{roomID}/start", api.startGame)
	mux.HandleFunc("POST /api/rooms/{roomID}/return-lobby", api.returnRoomToLobby)
	mux.HandleFunc("POST /api/rooms/{roomID}/rematch", api.returnRoomToLobby)
	mux.HandleFunc("POST /api/rooms/{roomID}/leave", api.leaveRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}/chat", api.recentChat)
	mux.HandleFunc("POST /api/rooms/{roomID}/chat", api.sendChat)
	mux.HandleFunc("POST /api/rooms/{roomID}/presence", api.markPresence)
	mux.HandleFunc("POST /api/reconnect", api.resumeConnection)
	mux.HandleFunc("POST /api/match/quick", api.quickMatch)
	mux.HandleFunc("GET /api/sessions/{sessionID}", api.getSession)
	mux.HandleFunc("POST /api/sessions/{sessionID}/actions", api.applyGameAction)
	mux.HandleFunc("GET /api/leaderboard", api.leaderboard)
	mux.HandleFunc("GET /api/games", api.listGames)
	mux.HandleFunc("GET /api/games/recommend", api.recommendGames)
	mux.HandleFunc("GET /ws", api.websocket)

	return withCORS(cfg, withRequestLog(logger, mux))
}
