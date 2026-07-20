package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"board-game-platform/apps/api/internal/auth"
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
	"board-game-platform/apps/api/internal/tutorial"
)

func NewRouter(cfg config.Config, logger *slog.Logger, games catalog.Catalog, auths *auth.Service, guests *guest.Service, rooms *room.Service, sessions *session.Service, chats *chat.Service, presence *connection.Service, records *record.Service, matches *match.Service, tutorials *tutorial.Service, hub *realtime.Hub, health HealthChecker) http.Handler {
	mux := http.NewServeMux()
	api := Handler{games: games, auths: auths, guests: guests, rooms: rooms, sessions: sessions, chats: chats, presence: presence, records: records, matches: matches, tutorials: tutorials, hub: hub, healthChecker: health}

	mux.HandleFunc("GET /health", api.health)
	mux.HandleFunc("POST /api/guests", api.createGuest)
	mux.HandleFunc("POST /api/auth/register", api.register)
	mux.HandleFunc("POST /api/auth/login", api.login)
	mux.HandleFunc("GET /api/auth/me", api.authMe)
	mux.HandleFunc("GET /api/me", api.me)
	mux.HandleFunc("POST /api/rooms", api.createRoom)
	mux.HandleFunc("POST /api/rooms/join", api.joinRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", api.getRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/ready", api.setReady)
	mux.HandleFunc("POST /api/rooms/{roomID}/spectate", api.spectateRoom)
	mux.HandleFunc("PATCH /api/rooms/{roomID}/options", api.updateRoomOptions)
	mux.HandleFunc("POST /api/rooms/{roomID}/kick", api.kickPlayer)
	mux.HandleFunc("POST /api/rooms/{roomID}/transfer-host", api.transferHost)
	mux.HandleFunc("POST /api/rooms/{roomID}/start", api.startGame)
	mux.HandleFunc("POST /api/rooms/{roomID}/return-lobby", api.returnRoomToLobby)
	mux.HandleFunc("POST /api/rooms/{roomID}/rematch", api.returnRoomToLobby)
	mux.HandleFunc("POST /api/rooms/{roomID}/leave", api.leaveRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}/chat", api.recentChat)
	mux.HandleFunc("POST /api/rooms/{roomID}/chat", api.sendChat)
	mux.HandleFunc("POST /api/rooms/{roomID}/presence", api.markPresence)
	mux.HandleFunc("POST /api/reconnect", api.resumeConnection)
	mux.HandleFunc("POST /api/match/quick", api.quickMatch)
	mux.HandleFunc("POST /api/match/cancel", api.cancelQuickMatch)
	mux.HandleFunc("GET /api/sessions/{sessionID}", api.getSession)
	mux.HandleFunc("POST /api/sessions/{sessionID}/actions", api.applyGameAction)
	mux.HandleFunc("GET /api/leaderboard", api.leaderboard)
	mux.HandleFunc("GET /api/games", api.listGames)
	mux.HandleFunc("GET /api/games/recommend", api.recommendGames)
	mux.HandleFunc("GET /api/tutorials", api.listTutorials)
	mux.HandleFunc("GET /api/tutorials/{gameID}", api.getTutorial)
	mux.HandleFunc("GET /ws", api.websocket)

	return withCORS(cfg, withRequestLog(logger, mux))
}

func NewPresenceSweeper(games catalog.Catalog, auths *auth.Service, guests *guest.Service, rooms *room.Service, sessions *session.Service, chats *chat.Service, presence *connection.Service, records *record.Service, matches *match.Service, tutorials *tutorial.Service, hub *realtime.Hub) func(context.Context) {
	api := Handler{games: games, auths: auths, guests: guests, rooms: rooms, sessions: sessions, chats: chats, presence: presence, records: records, matches: matches, tutorials: tutorials, hub: hub}
	return api.SweepExpiredPresence
}
