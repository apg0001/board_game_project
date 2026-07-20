package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/auth"
	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/chat"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/connection"
	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/match"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/record"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"

	"github.com/gorilla/websocket"
)

func TestRoomUpdatedBroadcastOnJoin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	hub := realtime.NewHub(logger)
	go hub.Run()

	server := httptest.NewServer(NewRouter(
		config.Config{HTTPAddr: ":0", AllowedOrigins: map[string]struct{}{}},
		logger,
		catalog.NewInMemoryCatalog(catalog.DefaultGames()),
		auth.NewService(nil),
		guest.NewService(guest.NewMemoryStore(), nil),
		room.NewService(room.NewMemoryStore(), nil),
		session.NewService(session.NewMemoryStore(), gamecore.NewRegistry(davinci.NewModule()), nil),
		chat.NewService(nil, 50),
		connection.NewService(nil, 0),
		record.NewService(nil),
		match.NewService(),
		hub,
	))
	defer server.Close()

	hostToken := createGuestTokenFromURL(t, server.URL)
	guestToken := createGuestTokenFromURL(t, server.URL)
	created := createRoomFromURL(t, server.URL, hostToken)

	socketURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?room=room:" + created.ID + "&user=host"
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	joinRoomFromURL(t, server.URL, guestToken, created.Code)

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	var message realtime.Message
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatal(err)
	}

	if message.Type != "room.updated" {
		t.Fatalf("expected room.updated, got %s", message.Type)
	}
}

func createGuestTokenFromURL(t *testing.T, baseURL string) string {
	t.Helper()

	response, err := http.Post(baseURL+"/api/guests", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", response.StatusCode)
	}

	var body struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.SessionToken
}

func createRoomFromURL(t *testing.T, baseURL string, token string) room.Room {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", response.StatusCode)
	}

	var body struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Room
}

func joinRoomFromURL(t *testing.T, baseURL string, token string, code string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, baseURL+"/api/rooms/join", bytes.NewBufferString(`{"code":"`+code+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
}
