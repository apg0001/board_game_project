package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"board-game-platform/apps/api/internal/auth"
	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/chat"
	"board-game-platform/apps/api/internal/config"
	"board-game-platform/apps/api/internal/connection"
	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/games/halligalli"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/match"
	"board-game-platform/apps/api/internal/realtime"
	"board-game-platform/apps/api/internal/record"
	"board-game-platform/apps/api/internal/room"
	"board-game-platform/apps/api/internal/session"
	"board-game-platform/apps/api/internal/tutorial"
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

func TestReturnLobbyAndLeaveRoom(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)
	room := createReadyRoom(t, handler, hostToken, guestToken)

	returnRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+room.ID+"/return-lobby", nil)
	returnRequest.Header.Set("Authorization", "Bearer "+hostToken)
	returnResponse := httptest.NewRecorder()
	handler.ServeHTTP(returnResponse, returnRequest)

	if returnResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", returnResponse.Code)
	}

	leaveRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+room.ID+"/leave", nil)
	leaveRequest.Header.Set("Authorization", "Bearer "+hostToken)
	leaveResponse := httptest.NewRecorder()
	handler.ServeHTTP(leaveResponse, leaveRequest)

	if leaveResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", leaveResponse.Code)
	}
}

func TestQuickMatchCreatesThenJoinsWaitingRoom(t *testing.T) {
	handler := testRouter()
	firstToken := createGuestToken(t, handler)
	secondToken := createGuestToken(t, handler)

	firstRequest := httptest.NewRequest(http.MethodPost, "/api/match/quick", bytes.NewBufferString(`{"gameId":"davinci"}`))
	firstRequest.Header.Set("Authorization", "Bearer "+firstToken)
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, firstRequest)
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", firstResponse.Code)
	}

	var firstBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(firstResponse.Body).Decode(&firstBody); err != nil {
		t.Fatal(err)
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/api/match/quick", bytes.NewBufferString(`{"gameId":"davinci"}`))
	secondRequest.Header.Set("Authorization", "Bearer "+secondToken)
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", secondResponse.Code)
	}

	var secondBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(secondResponse.Body).Decode(&secondBody); err != nil {
		t.Fatal(err)
	}
	if firstBody.Room.ID != secondBody.Room.ID {
		t.Fatalf("expected same room, got %s and %s", firstBody.Room.ID, secondBody.Room.ID)
	}
	if len(secondBody.Room.Participants) != 2 {
		t.Fatalf("expected two participants, got %d", len(secondBody.Room.Participants))
	}
}

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	registry := gamecore.NewRegistry(davinci.NewModule(), halligalli.NewModule())
	return NewRouter(
		config.Config{HTTPAddr: ":0", AllowedOrigins: map[string]struct{}{"http://localhost:5173": {}}},
		logger,
		catalog.NewInMemoryCatalog(catalog.DefaultGames()),
		auth.NewService(nil),
		guest.NewService(guest.NewMemoryStore(), nil),
		room.NewService(room.NewMemoryStore(), nil),
		session.NewService(session.NewMemoryStore(), registry, nil),
		chat.NewService(nil, 50),
		connection.NewService(nil, 0),
		record.NewService(nil),
		match.NewService(),
		tutorial.NewService(),
		realtime.NewHub(logger),
	)
}

func TestRegisterAndLogin(t *testing.T) {
	handler := testRouter()

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"alice","password":"secret","nickname":"Alice"}`))
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", registerResponse.Code)
	}

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"alice","password":"secret"}`))
	loginResponse := httptest.NewRecorder()
	handler.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", loginResponse.Code)
	}
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
