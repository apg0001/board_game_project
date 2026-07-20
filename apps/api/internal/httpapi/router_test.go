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
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/realtime"
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

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	return NewRouter(
		config.Config{HTTPAddr: ":0", AllowedOrigins: map[string]struct{}{"http://localhost:5173": {}}},
		logger,
		catalog.NewInMemoryCatalog(catalog.DefaultGames()),
		guest.NewService(guest.NewMemoryStore(), nil),
		realtime.NewHub(logger),
	)
}
