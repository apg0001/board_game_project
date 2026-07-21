package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"board-game-platform/apps/api/internal/auth"
	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/chat"
	"board-game-platform/apps/api/internal/connection"
	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/match"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/record"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"
	"board-game-platform/apps/api/internal/tutorial"
)

type Handler struct {
	games         catalog.Catalog
	auths         *auth.Service
	guests        *guest.Service
	rooms         *room.Service
	sessions      *session.Service
	chats         *chat.Service
	presence      *connection.Service
	records       *record.Service
	matches       *match.Service
	tutorials     *tutorial.Service
	hub           *realtime.Hub
	healthChecker HealthChecker
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

func (h Handler) health(w http.ResponseWriter, _ *http.Request) {
	database := "disabled"
	if h.healthChecker != nil {
		database = "ok"
		if err := h.healthChecker.Ping(context.Background()); err != nil {
			database = "unhealthy"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"runtime": map[string]any{
			"presenceGraceSeconds": int(h.presence.GracePeriod().Seconds()),
			"games":                len(h.games.All()),
			"database":             database,
		},
	})
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

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid register payload"})
		return
	}
	user, token, err := h.auths.Register(body.Username, body.Password, body.Nickname)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, auth.ErrDuplicateUsername) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": user, "sessionToken": token})
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid login payload"})
		return
	}
	user, token, err := h.auths.Login(body.Username, body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "sessionToken": token})
}

func (h Handler) authMe(w http.ResponseWriter, r *http.Request) {
	user, ok := h.auths.Me(bearerToken(r))
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid auth session"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
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

func (h Handler) createRoom(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	var body struct {
		GameID     string `json:"gameId"`
		MaxPlayers int    `json:"maxPlayers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && r.ContentLength != 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid room payload"})
		return
	}

	created, err := h.rooms.Create(user.Public(), body.GameID, body.MaxPlayers)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create room"})
		return
	}

	h.publishRoomUpdated(created)
	writeJSON(w, http.StatusCreated, map[string]any{"room": created})
}

func (h Handler) joinRoom(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room code is required"})
		return
	}

	joined, err := h.rooms.JoinByCode(body.Code, user.Public())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, room.ErrRoomNotFound) {
			status = http.StatusNotFound
		}
		if errors.Is(err, room.ErrRoomFull) || errors.Is(err, room.ErrRoomNotJoinable) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	h.publishRoomUpdated(joined)
	writeJSON(w, http.StatusOK, map[string]any{"room": joined})
}

func (h Handler) getRoom(w http.ResponseWriter, r *http.Request) {
	found, err := h.rooms.FindByID(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"room": found})
}

func (h Handler) setReady(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	var body struct {
		Ready bool `json:"ready"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ready payload"})
		return
	}

	updated, err := h.rooms.ToggleReady(r.PathValue("roomID"), user.ID, body.Ready)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room participant not found"})
		return
	}

	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) spectateRoom(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	updated, err := h.rooms.JoinSpectator(r.PathValue("roomID"), user.Public())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) updateRoomOptions(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	found, err := h.rooms.FindByID(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}
	if found.HostUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only host can update options"})
		return
	}

	var body struct {
		TurnSeconds int `json:"turnSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid options payload"})
		return
	}

	updated, err := h.rooms.UpdateOptions(found.ID, body.TurnSeconds)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update options"})
		return
	}
	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) kickPlayer(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	found, err := h.rooms.FindByID(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}
	if found.HostUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only host can kick"})
		return
	}
	var body struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target user is required"})
		return
	}
	updated, err := h.rooms.Kick(found.ID, body.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target not found"})
		return
	}
	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) transferHost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	found, err := h.rooms.FindByID(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}
	if found.HostUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only host can transfer host"})
		return
	}
	var body struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "next host is required"})
		return
	}
	updated, err := h.rooms.TransferHost(found.ID, body.UserID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "next host not found"})
		return
	}
	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) startGame(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	var body struct {
		PlayerIDs []string `json:"playerIds"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	found, err := h.rooms.FindByID(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}
	if found.HostUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only host can start game"})
		return
	}

	created, err := h.sessions.StartWithPlayers(found, body.PlayerIDs)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, session.ErrRoomNotReady) || errors.Is(err, session.ErrRoomOverCapacity) || errors.Is(err, session.ErrGameNotRegistered) {
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	playingIDs := body.PlayerIDs
	if len(playingIDs) == 0 {
		playingIDs = participantIDs(found.Participants)
	}
	updatedRoom, err := h.rooms.SetPlaying(found.ID, created.ID, playingIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update room status"})
		return
	}

	publicSession, err := h.sessions.PublicView(updatedRoom, created, gamecore.PlayerID(user.ID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build session view"})
		return
	}

	h.publishRoomUpdated(updatedRoom)
	h.publishGameUpdated(created)
	writeJSON(w, http.StatusCreated, map[string]any{"session": publicSession, "room": updatedRoom})
}

func (h Handler) returnRoomToLobby(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireGuest(w, r); !ok {
		return
	}

	updated, err := h.rooms.ReturnToLobby(r.PathValue("roomID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) leaveRoom(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	updated, err := h.rooms.Leave(r.PathValue("roomID"), user.ID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	h.publishRoomUpdated(updated)
	writeJSON(w, http.StatusOK, map[string]any{"room": updated})
}

func (h Handler) recentChat(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"messages": h.chats.Recent(r.PathValue("roomID"))})
}

func (h Handler) sendChat(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	var body struct {
		Text string `json:"text"`
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid chat payload"})
		return
	}
	message, saved := h.chats.Add(r.PathValue("roomID"), user.Public(), body.Text, body.Kind)
	if !saved {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is empty"})
		return
	}
	h.hub.Broadcast(realtime.Message{Room: "room:" + r.PathValue("roomID"), Type: "chat.message", Payload: map[string]any{"message": message}})
	writeJSON(w, http.StatusCreated, map[string]any{"message": message})
}

func (h Handler) markPresence(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	var body struct {
		SessionID string `json:"sessionId"`
		Status    string `json:"status"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	var presence connection.Presence
	if body.Status == string(connection.StatusDisconnected) {
		var found bool
		presence, found = h.presence.MarkDisconnected(user.ID)
		if !found {
			presence = h.presence.MarkOnline(user.ID, r.PathValue("roomID"), body.SessionID)
			presence, _ = h.presence.MarkDisconnected(user.ID)
		}
	} else {
		presence = h.presence.MarkOnline(user.ID, r.PathValue("roomID"), body.SessionID)
	}
	h.hub.Broadcast(realtime.Message{Room: "room:" + r.PathValue("roomID"), Type: "presence.updated", Payload: map[string]any{"presence": presence}})
	writeJSON(w, http.StatusOK, map[string]any{"presence": presence})
}

func (h Handler) resumeConnection(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	presence, resumed := h.presence.Resume(user.ID)
	if !resumed {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "reconnect window expired"})
		return
	}
	h.hub.Broadcast(realtime.Message{Room: "room:" + presence.RoomID, Type: "presence.updated", Payload: map[string]any{"presence": presence}})
	writeJSON(w, http.StatusOK, map[string]any{"presence": presence})
}

func (h Handler) SweepExpiredPresence(ctx context.Context) {
	for _, expired := range h.presence.ExpireDisconnected() {
		h.hub.Broadcast(realtime.Message{
			Room:    "room:" + expired.RoomID,
			Type:    "presence.expired",
			Payload: map[string]any{"presence": expired},
		})
		if expired.RoomID == "" || expired.SessionID == "" {
			continue
		}

		foundRoom, err := h.rooms.FindByID(expired.RoomID)
		if err != nil || foundRoom.ActiveSessionID != expired.SessionID {
			continue
		}
		updated, events, err := h.sessions.ApplyTimeout(ctx, foundRoom, expired.SessionID, gamecore.PlayerID(expired.UserID))
		if err != nil {
			continue
		}
		if updated.Status == session.StatusFinished {
			h.records.RecordSession(updated)
			if finishedRoom, statusErr := h.rooms.SetStatus(foundRoom.ID, room.StatusFinished); statusErr == nil {
				h.publishRoomUpdated(finishedRoom)
			}
		}
		h.publishGameUpdated(updated)
		for _, event := range events {
			h.hub.Broadcast(realtime.Message{
				Room:    "game:" + updated.ID,
				Type:    event.Type,
				Payload: event.Payload,
			})
		}
	}
}

func (h Handler) quickMatch(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	var body struct {
		GameID     string `json:"gameId"`
		MaxPlayers int    `json:"maxPlayers"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.GameID == "" {
		body.GameID = "davinci"
	}

	for {
		roomID, found := h.matches.NextWaiting(body.GameID)
		if !found {
			created, err := h.rooms.Create(user.Public(), body.GameID, body.MaxPlayers)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create match room"})
				return
			}
			h.matches.AddWaiting(body.GameID, created.ID)
			h.publishRoomUpdated(created)
			writeJSON(w, http.StatusCreated, map[string]any{"room": created, "matched": false})
			return
		}

		waiting, err := h.rooms.FindByID(roomID)
		if err != nil || waiting.Status != room.StatusLobby || waiting.IsFull() {
			continue
		}
		joined, err := h.rooms.JoinByCode(waiting.Code, user.Public())
		if err != nil {
			continue
		}
		if !joined.IsFull() {
			h.matches.AddWaiting(body.GameID, joined.ID)
		}
		h.publishRoomUpdated(joined)
		writeJSON(w, http.StatusOK, map[string]any{"room": joined, "matched": true})
		return
	}
}

func (h Handler) cancelQuickMatch(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}
	var body struct {
		RoomID string `json:"roomId"`
		GameID string `json:"gameId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RoomID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room is required"})
		return
	}
	found, err := h.rooms.FindByID(body.RoomID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}
	if found.HostUserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "only host can cancel quick match"})
		return
	}
	gameID := body.GameID
	if gameID == "" {
		gameID = found.GameID
	}
	cancelled := h.matches.Cancel(gameID, found.ID)
	writeJSON(w, http.StatusOK, map[string]any{"cancelled": cancelled})
}

func (h Handler) getSession(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	found, err := h.sessions.FindByID(r.PathValue("sessionID"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}

	foundRoom, err := h.rooms.FindByID(found.RoomID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	publicSession, err := h.sessions.PublicView(foundRoom, found, gamecore.PlayerID(user.ID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build session view"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"session": publicSession})
}

func (h Handler) applyGameAction(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireGuest(w, r)
	if !ok {
		return
	}

	var body struct {
		RoomID          string `json:"roomId"`
		Type            string `json:"type"`
		Payload         any    `json:"payload"`
		ClientRequestID string `json:"clientRequestId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action payload"})
		return
	}

	foundRoom, err := h.rooms.FindByID(body.RoomID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "room not found"})
		return
	}

	updated, events, err := h.sessions.ApplyAction(r.Context(), foundRoom, r.PathValue("sessionID"), gamecore.Action{
		Type:            body.Type,
		PlayerID:        gamecore.PlayerID(user.ID),
		Payload:         body.Payload,
		ClientRequestID: body.ClientRequestID,
		CreatedAt:       timeNowUTC(),
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	publicSession, err := h.sessions.PublicView(foundRoom, updated, gamecore.PlayerID(user.ID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build session view"})
		return
	}

	if updated.Status == session.StatusFinished {
		h.records.RecordSession(updated)
		if finishedRoom, statusErr := h.rooms.SetStatus(foundRoom.ID, room.StatusFinished); statusErr == nil {
			h.publishRoomUpdated(finishedRoom)
		}
	}

	h.publishGameUpdated(updated)
	for _, event := range events {
		h.hub.Broadcast(realtime.Message{
			Room:    "game:" + updated.ID,
			Type:    event.Type,
			Payload: event.Payload,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": publicSession})
}

func (h Handler) leaderboard(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, map[string]any{"rows": h.records.Leaderboard(r.URL.Query().Get("gameId"), limit)})
}

func (h Handler) listTutorials(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"guides": h.tutorials.All()})
}

func (h Handler) getTutorial(w http.ResponseWriter, r *http.Request) {
	guide, ok := h.tutorials.Find(r.PathValue("gameID"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "tutorial not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"guide": guide})
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

func (h Handler) publishRoomUpdated(updated room.Room) {
	h.hub.Broadcast(realtime.Message{
		Room: "room:" + updated.ID,
		Type: "room.updated",
		Payload: map[string]any{
			"room": updated,
		},
	})
}

func (h Handler) publishGameUpdated(updated session.Session) {
	foundRoom, err := h.rooms.FindByID(updated.RoomID)
	if err == nil {
		publicSession, viewErr := h.sessions.PublicView(foundRoom, updated, "")
		if viewErr == nil {
			updated = publicSession
		}
	}

	h.hub.Broadcast(realtime.Message{
		Room: "game:" + updated.ID,
		Type: "game.updated",
		Payload: map[string]any{
			"session": updated,
		},
	})
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

func participantIDs(participants []room.Participant) []string {
	ids := make([]string, 0, len(participants))
	for _, participant := range participants {
		ids = append(ids, participant.User.ID)
	}
	return ids
}

func (h Handler) requireGuest(w http.ResponseWriter, r *http.Request) (guest.User, bool) {
	user, err := h.guests.Me(bearerToken(r))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "valid guest session is required"})
		return guest.User{}, false
	}
	return user, true
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
