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
	"board-game-platform/apps/api/internal/games/bang"
	"board-game-platform/apps/api/internal/games/dalmuti"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/games/gostop"
	"board-game-platform/apps/api/internal/games/halligalli"
	"board-game-platform/apps/api/internal/games/jokerdraw"
	"board-game-platform/apps/api/internal/games/onecard"
	"board-game-platform/apps/api/internal/games/rummikub"
	"board-game-platform/apps/api/internal/games/splendor"
	"board-game-platform/apps/api/internal/games/sutda"
	"board-game-platform/apps/api/internal/games/werewolf"
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
		bytes.NewBufferString(`{"roomId":"`+room.ID+`","type":"davinci.guess","payload":{"targetPlayerId":"`+room.Participants[1].User.ID+`","tileIndex":0,"color":"black","value":0},"clientRequestId":"test-1"}`),
	)
	actionRequest.Header.Set("Authorization", "Bearer "+hostToken)
	actionResponse := httptest.NewRecorder()
	handler.ServeHTTP(actionResponse, actionRequest)

	if actionResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", actionResponse.Code)
	}
}

func TestOneCardRuleVoteResolvesOnStart(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"onecard","maxPlayers":4}`))
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

	voteRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/rules/vote", bytes.NewBufferString(`{"choices":{"attackCards":"two-ace-joker","defenseMode":"attack-or-joker","jokerDrawCount":"7","stacking":"on"}}`))
	voteRequest.Header.Set("Authorization", "Bearer "+hostToken)
	voteResponse := httptest.NewRecorder()
	handler.ServeHTTP(voteResponse, voteRequest)
	if voteResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", voteResponse.Code)
	}
	var voteBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(voteResponse.Body).Decode(&voteBody); err != nil {
		t.Fatal(err)
	}
	if len(voteBody.Room.RuleVotes) != 1 {
		t.Fatalf("expected one rule vote, got %+v", voteBody.Room.RuleVotes)
	}

	setReadyForTest(t, handler, createBody.Room.ID, hostToken)
	setReadyForTest(t, handler, createBody.Room.ID, guestToken)

	startRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/start", nil)
	startRequest.Header.Set("Authorization", "Bearer "+hostToken)
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, startRequest)
	if startResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", startResponse.Code)
	}
	var startBody struct {
		Room    room.Room       `json:"room"`
		Session session.Session `json:"session"`
	}
	if err := json.NewDecoder(startResponse.Body).Decode(&startBody); err != nil {
		t.Fatal(err)
	}
	if startBody.Room.GameRules["jokerDrawCount"] == nil || len(startBody.Room.RuleMessages) == 0 {
		t.Fatalf("expected resolved onecard rules, got rules=%+v messages=%+v", startBody.Room.GameRules, startBody.Room.RuleMessages)
	}
	state, ok := startBody.Session.State.(map[string]any)
	if !ok || state["rules"] == nil {
		t.Fatalf("expected onecard rules in session state, got %#v", startBody.Session.State)
	}
}

func TestStartGameWithSelectedPlayersLeavesOverflowAsSpectators(t *testing.T) {
	handler := testRouter()
	tokens := []string{
		createGuestToken(t, handler),
		createGuestToken(t, handler),
		createGuestToken(t, handler),
		createGuestToken(t, handler),
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"gostop","maxPlayers":12}`))
	createRequest.Header.Set("Authorization", "Bearer "+tokens[0])
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
	for _, token := range tokens[1:] {
		joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/join", bytes.NewBufferString(`{"code":"`+createBody.Room.Code+`"}`))
		joinRequest.Header.Set("Authorization", "Bearer "+token)
		joinResponse := httptest.NewRecorder()
		handler.ServeHTTP(joinResponse, joinRequest)
		if joinResponse.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", joinResponse.Code)
		}
	}
	for _, token := range tokens[:3] {
		setReadyForTest(t, handler, createBody.Room.ID, token)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/rooms/"+createBody.Room.ID, nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	var getBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(getResponse.Body).Decode(&getBody); err != nil {
		t.Fatal(err)
	}
	selected := []string{
		getBody.Room.Participants[0].User.ID,
		getBody.Room.Participants[1].User.ID,
		getBody.Room.Participants[2].User.ID,
	}
	startBody, err := json.Marshal(map[string]any{"playerIds": selected})
	if err != nil {
		t.Fatal(err)
	}
	startRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/start", bytes.NewReader(startBody))
	startRequest.Header.Set("Authorization", "Bearer "+tokens[0])
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, startRequest)
	if startResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", startResponse.Code)
	}
	var body struct {
		Room    room.Room       `json:"room"`
		Session session.Session `json:"session"`
	}
	if err := json.NewDecoder(startResponse.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Room.Participants) != 4 || len(body.Room.PlayingPlayerIDs) != 3 {
		t.Fatalf("expected four lobby participants and three playing ids, got %+v", body.Room)
	}
	if len(body.Session.Results) != 0 || body.Session.GameID != "gostop" {
		t.Fatalf("unexpected session: %+v", body.Session)
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

func TestSpectateAndUpdateRoomOptions(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	spectatorToken := createGuestToken(t, handler)
	blockedSpectatorToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4}`))
	createRequest.Header.Set("Authorization", "Bearer "+hostToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	var createBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}

	spectateRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/spectate", nil)
	spectateRequest.Header.Set("Authorization", "Bearer "+spectatorToken)
	spectateResponse := httptest.NewRecorder()
	handler.ServeHTTP(spectateResponse, spectateRequest)
	if spectateResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", spectateResponse.Code)
	}

	optionsRequest := httptest.NewRequest(http.MethodPatch, "/api/rooms/"+createBody.Room.ID+"/options", bytes.NewBufferString(`{"turnSeconds":45,"maxWaitSeconds":240,"autoStart":true,"allowSpectators":false,"maxPlayers":6}`))
	optionsRequest.Header.Set("Authorization", "Bearer "+hostToken)
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", optionsResponse.Code)
	}
	var optionsBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(optionsResponse.Body).Decode(&optionsBody); err != nil {
		t.Fatal(err)
	}
	if optionsBody.Room.Options.TurnSeconds != 45 || optionsBody.Room.Options.MaxWaitSeconds != 240 {
		t.Fatalf("unexpected options: %+v", optionsBody.Room.Options)
	}
	if optionsBody.Room.MaxPlayers != 6 {
		t.Fatalf("expected max players updated, got %d", optionsBody.Room.MaxPlayers)
	}
	if !optionsBody.Room.Options.AutoStart || optionsBody.Room.Options.AllowSpectators {
		t.Fatalf("unexpected boolean options: %+v", optionsBody.Room.Options)
	}

	blockedSpectateRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/spectate", nil)
	blockedSpectateRequest.Header.Set("Authorization", "Bearer "+spectatorToken)
	blockedSpectateResponse := httptest.NewRecorder()
	handler.ServeHTTP(blockedSpectateResponse, blockedSpectateRequest)
	if blockedSpectateResponse.Code != http.StatusOK {
		t.Fatalf("existing spectator should remain in room, got %d", blockedSpectateResponse.Code)
	}

	newSpectateRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/spectate", nil)
	newSpectateRequest.Header.Set("Authorization", "Bearer "+blockedSpectatorToken)
	newSpectateResponse := httptest.NewRecorder()
	handler.ServeHTTP(newSpectateResponse, newSpectateRequest)
	if newSpectateResponse.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for blocked spectator, got %d", newSpectateResponse.Code)
	}
}

func TestListAndJoinPublicRoom(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"splendor","maxPlayers":4,"visibility":"PUBLIC"}`))
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
	if createBody.Room.Visibility != room.VisibilityPublic {
		t.Fatalf("expected public room, got %s", createBody.Room.Visibility)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/rooms?gameId=splendor", nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResponse.Code)
	}
	var listBody struct {
		Rooms []room.Room `json:"rooms"`
	}
	if err := json.NewDecoder(listResponse.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	if len(listBody.Rooms) != 1 || listBody.Rooms[0].ID != createBody.Room.ID {
		t.Fatalf("unexpected public room list: %+v", listBody.Rooms)
	}

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/join", nil)
	joinRequest.Header.Set("Authorization", "Bearer "+guestToken)
	joinResponse := httptest.NewRecorder()
	handler.ServeHTTP(joinResponse, joinRequest)
	if joinResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", joinResponse.Code)
	}
	var joinBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(joinResponse.Body).Decode(&joinBody); err != nil {
		t.Fatal(err)
	}
	if len(joinBody.Room.Participants) != 2 {
		t.Fatalf("expected joined participant, got %+v", joinBody.Room.Participants)
	}
}

func TestJoinPlayingPublicRoomBecomesSpectator(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)
	spectatorToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4,"visibility":"PUBLIC"}`))
	createRequest.Header.Set("Authorization", "Bearer "+hostToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	var createBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/join", nil)
	joinRequest.Header.Set("Authorization", "Bearer "+guestToken)
	joinResponse := httptest.NewRecorder()
	handler.ServeHTTP(joinResponse, joinRequest)
	setReadyForTest(t, handler, createBody.Room.ID, hostToken)
	setReadyForTest(t, handler, createBody.Room.ID, guestToken)

	startRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/start", bytes.NewBufferString(`{"playerIds":[]}`))
	startRequest.Header.Set("Authorization", "Bearer "+hostToken)
	startResponse := httptest.NewRecorder()
	handler.ServeHTTP(startResponse, startRequest)
	if startResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", startResponse.Code)
	}

	spectatorJoinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createBody.Room.ID+"/join", nil)
	spectatorJoinRequest.Header.Set("Authorization", "Bearer "+spectatorToken)
	spectatorJoinResponse := httptest.NewRecorder()
	handler.ServeHTTP(spectatorJoinResponse, spectatorJoinRequest)
	if spectatorJoinResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", spectatorJoinResponse.Code)
	}
	var spectatorBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(spectatorJoinResponse.Body).Decode(&spectatorBody); err != nil {
		t.Fatal(err)
	}
	if len(spectatorBody.Room.Spectators) != 1 {
		t.Fatalf("expected spectator join, got %+v", spectatorBody.Room.Spectators)
	}
}

func TestAutoStartWhenAllParticipantsReady(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":4}`))
	createRequest.Header.Set("Authorization", "Bearer "+hostToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	var createBody struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(createResponse.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}

	optionsRequest := httptest.NewRequest(http.MethodPatch, "/api/rooms/"+createBody.Room.ID+"/options", bytes.NewBufferString(`{"turnSeconds":60,"maxWaitSeconds":180,"autoStart":true,"allowSpectators":true}`))
	optionsRequest.Header.Set("Authorization", "Bearer "+hostToken)
	optionsResponse := httptest.NewRecorder()
	handler.ServeHTTP(optionsResponse, optionsRequest)
	if optionsResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", optionsResponse.Code)
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
	if getBody.Room.Status != room.StatusPlaying || getBody.Room.ActiveSessionID == "" {
		t.Fatalf("expected auto started room, got %+v", getBody.Room)
	}
}

func TestKickAndTransferHost(t *testing.T) {
	handler := testRouter()
	hostToken := createGuestToken(t, handler)
	guestToken := createGuestToken(t, handler)
	createdRoom := createReadyRoom(t, handler, hostToken, guestToken)

	var roomBody struct {
		Room room.Room `json:"room"`
	}
	getRequest := httptest.NewRequest(http.MethodGet, "/api/rooms/"+createdRoom.ID, nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if err := json.NewDecoder(getResponse.Body).Decode(&roomBody); err != nil {
		t.Fatal(err)
	}
	targetID := roomBody.Room.Participants[1].User.ID

	transferRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createdRoom.ID+"/transfer-host", bytes.NewBufferString(`{"userId":"`+targetID+`"}`))
	transferRequest.Header.Set("Authorization", "Bearer "+hostToken)
	transferResponse := httptest.NewRecorder()
	handler.ServeHTTP(transferResponse, transferRequest)
	if transferResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", transferResponse.Code)
	}

	kickRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+createdRoom.ID+"/kick", bytes.NewBufferString(`{"userId":"`+roomBody.Room.Participants[0].User.ID+`"}`))
	kickRequest.Header.Set("Authorization", "Bearer "+guestToken)
	kickResponse := httptest.NewRecorder()
	handler.ServeHTTP(kickResponse, kickRequest)
	if kickResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", kickResponse.Code)
	}
}

func TestCancelQuickMatch(t *testing.T) {
	handler := testRouter()
	token := createGuestToken(t, handler)
	request := httptest.NewRequest(http.MethodPost, "/api/match/quick", bytes.NewBufferString(`{"gameId":"davinci"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	var body struct {
		Room room.Room `json:"room"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	cancelRequest := httptest.NewRequest(http.MethodPost, "/api/match/cancel", bytes.NewBufferString(`{"roomId":"`+body.Room.ID+`","gameId":"davinci"}`))
	cancelRequest.Header.Set("Authorization", "Bearer "+token)
	cancelResponse := httptest.NewRecorder()
	handler.ServeHTTP(cancelResponse, cancelRequest)
	if cancelResponse.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", cancelResponse.Code)
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
	registry := gamecore.NewRegistry(bang.NewModule(), dalmuti.NewModule(), davinci.NewModule(), gostop.NewModule(), halligalli.NewModule(), jokerdraw.NewModule(), onecard.NewModule(), rummikub.NewModule(), splendor.NewModule(), sutda.NewModule(), werewolf.NewModule())
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
		nil,
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

func TestRegisteredUserCanCreateRoom(t *testing.T) {
	handler := testRouter()

	registerRequest := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(`{"username":"bob","password":"secret","nickname":"Bob"}`))
	registerResponse := httptest.NewRecorder()
	handler.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", registerResponse.Code)
	}

	var registerBody struct {
		SessionToken string `json:"sessionToken"`
	}
	if err := json.NewDecoder(registerResponse.Body).Decode(&registerBody); err != nil {
		t.Fatal(err)
	}

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewBufferString(`{"gameId":"davinci","maxPlayers":12}`))
	createRequest.Header.Set("Authorization", "Bearer "+registerBody.SessionToken)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResponse.Code)
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
