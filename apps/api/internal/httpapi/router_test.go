package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"board-game-platform/apps/api/internal/catalog"
	"board-game-platform/apps/api/internal/config"
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

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(httptest.NewRecorder(), nil))
	return NewRouter(
		config.Config{HTTPAddr: ":0", AllowedOrigins: map[string]struct{}{"http://localhost:5173": {}}},
		logger,
		catalog.NewInMemoryCatalog(catalog.DefaultGames()),
		realtime.NewHub(logger),
	)
}
