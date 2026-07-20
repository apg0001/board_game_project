package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/realtime"
)

type Handler struct {
	games catalog.Catalog
	hub   *realtime.Hub
}

func (h Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) listGames(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"games": h.games.All()})
}

func (h Handler) recommendGames(w http.ResponseWriter, r *http.Request) {
	playerCount, err := strconv.Atoi(r.URL.Query().Get("players"))
	if err != nil || playerCount < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "players must be a positive number"})
		return
	}

	category := catalog.Category(r.URL.Query().Get("category"))
	writeJSON(w, http.StatusOK, map[string]any{
		"playerCount": playerCount,
		"games":       h.games.Recommend(playerCount, category),
	})
}

func (h Handler) websocket(w http.ResponseWriter, r *http.Request) {
	realtime.ServeWebSocket(h.hub, w, r)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
