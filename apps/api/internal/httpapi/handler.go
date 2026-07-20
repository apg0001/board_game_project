package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/realtime"
)

type Handler struct {
	games  catalog.Catalog
	guests *guest.Service
	hub    *realtime.Hub
}

func (h Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h Handler) createGuest(w http.ResponseWriter, _ *http.Request) {
	user, err := h.guests.Create()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create guest"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":         user.Public(),
		"sessionToken": user.SessionToken,
	})
}

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.guests.Me(bearerToken(r))
	if err != nil {
		if errors.Is(err, guest.ErrSessionNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid guest session"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load session"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user.Public()})
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

func bearerToken(r *http.Request) string {
	value := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(value, "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}
