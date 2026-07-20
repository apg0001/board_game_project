package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"
)

func TestHealth(t *testing.T) {
	handler := testRouter()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
}

func TestRecommendGames(t *testing.T) {
	handler := testRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/games/recommend?players=4", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}

	var body struct {
		PlayerCount int            `json:"playerCount"`
		Games       []catalog.Game `json:"games"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PlayerCount != 4 || len(body.Games) == 0 {
		t.Fatalf("unexpected recommendation body: %+v", body)
	}
}

func TestCreateGuestAndMe(t *testing.T) {
	handler := testRouter()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/guests", bytes.NewReader(nil))
	createResponse := httptest.NewRecorder()

	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResponse.Code)
	}

	var createBody struct {
		SessionToken string `json:"sessionToken"`
		User         struct {
			Nickname string `json:"nickname"`
		} `json:"user"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}
	if createBody.SessionToken == "" || createBody.User.Nickname == "" {
		t.Fatalf("unexpected guest response: %+v", createBody)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+createBody.SessionToken)
	meResponse := httptest.NewRecorder()

	handler.ServeHTTP(meResponse, meRequest)

	if meResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", meResponse.Code)
	}
}

func TestMeRejectsMissingToken(t *testing.T) {
	handler := testRouter()
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestCreateJoinAndReadyRoom(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4}`))
	createRequest.Header.Set("Authorization", "Bearer "+hostToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResponse.Code)
	}

	var createBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}
	if createBody.Room.Code == "" {
		t.Fatal("expected room code")
	}

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewBufferString(`{"code":"`+createBody.Room.Code+`"}`))
	joinRequest.Header.Set("Authorization", "Bearer "+guestToken)
	joinResponse := httptest.NewRecorder()
	handler.ServeHTTP(joinResponse, joinRequest)

	if joinResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", joinResponse.Code)
	}

	readyRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/ready", bytes.NewBufferString(`{"ready":true}`))
	readyRequest.Header.Set("Authorization", "Bearer "+hostToken)
	readyResponse := httptest.NewRecorder()
	handler.ServeHTTP(readyResponse, readyRequest)

	if readyResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", readyResponse.Code)
	}
}

func TestStartGameAndApplyAction(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)

	room := createReadyRoom(t, handler, hostToken, guestToken)

	startRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+room.ID+"/start", nil)
	startRequest.Header.Set("Authorization", "Bearer "+hostToken)
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, startRequest)

	if startResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", startResponse.Code)
	}

	var startBody struct {
		Session session.Session `json:"session"`
	}
	if err := json.NewDecoder(startResponse.Body).Decode(&startBody); err != nil {
		t.Fatal(err)
	}

	actionRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/"+startBody.Session.ID+"/actions",
		bytes.NewBufferString(`{"roomId":"`+room.ID+`","type":"davinci.pass","clientRequestId":"test-1"}`),
	)
	actionRequest.Header.Set("Authorization", "Bearer "+hostToken)
	actionResponse := httptest.NewRecorder()
	handler.ServeHTTP(actionResponse, actionRequest)

	if actionResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", actionResponse.Code)
	}
}

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	registry := gamecore.NewRegistry(davinci.NewModule())
	return NewRouter(
		config.Config{HTTPAddr: ":0", AllowedOrigins: map[string]struct{}{"http://localhost:5173": {}}},
		logger,
		catalog.NewInMemoryCatalog(catalog.DefaultGames()),
		guest.NewService(guest.NewMemoryStore(), nil),
		room.NewService(room.NewMemoryStore(), nil),
		session.NewService(session.NewMemoryStore(), registry, nil),
		realtime.NewHub(logger),
	)
}

func createGuestToken(t *testing.T, handler http.Handler) string {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/guests", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", response.Code)
	}

	var body struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.SessionToken
}

func createReadyRoom(t *testing.T, handler http.Handler, hostToken string, guestToken string) room.Room {
	t.Helper()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4}`))
	createRequest.Header.Set("Authorization", "Bearer "+hostToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResponse.Code)
	}

	var createBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewBufferString(`{"code":"`+createBody.Room.Code+`"}`))
	joinRequest.Header.Set("Authorization", "Bearer "+guestToken)
	joinResponse := httptest.NewRecorder()
	handler.ServeHTTP(joinResponse, joinRequest)

	if joinResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", joinResponse.Code)
	}

	setReadyForTest(t, handler, createBody.Room.ID, hostToken)
	setReadyForTest(t, handler, createBody.Room.ID, guestToken)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/rooms/"+createBody.Room.ID, nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)

	var getBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(getResponse.Body).Decode(&getBody); err != nil {
		t.Fatal(err)
	}
	return getBody.Room
}

func setReadyForTest(t *testing.T, handler http.Handler, roomID string, token string) {
	t.Helper()

	readyRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/ready", bytes.NewBufferString(`{"ready":true}`))
	readyRequest.Header.Set("Authorization", "Bearer "+token)
	readyResponse := httptest.NewRecorder()
	handler.ServeHTTP(readyResponse, readyRequest)

	if readyResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", readyResponse.Code)
	}
}
