package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"board-game-platform/apps/api/internal/guest"
)

var (
	ErrRoomFull        = errors.New("room is full")
	ErrRoomNotJoinable = errors.New("room is not joinable")
	ErrSpectatorClosed = errors.New("spectator mode is disabled")
)

type Clock func() time.Time

type Service struct {
	store Store
	clock Clock
}

type CreateOptions struct {
	GameID     string
	MaxPlayers int
	Visibility Visibility
}

type UpdateOptionsRequest struct {
	Options    Options
	MaxPlayers int
}

type ListFilter struct {
	GameID        string
	Visibility    Visibility
	IncludeClosed bool
}

func NewService(store Store, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: store, clock: clock}
}

func (s *Service) Create(host guest.PublicUser, gameID string, maxPlayers int) (Room, error) {
	return s.CreateWithOptions(host, CreateOptions{GameID: gameID, MaxPlayers: maxPlayers, Visibility: VisibilityPrivate})
}

func (s *Service) CreateWithOptions(host guest.PublicUser, options CreateOptions) (Room, error) {
	host = normalizePublicUser(host, "host")
	gameID := options.GameID
	if gameID == "" {
		gameID = "davinci"
	}
	maxPlayers := options.MaxPlayers
	if maxPlayers <= 0 {
		maxPlayers = 4
	}
	visibility := options.Visibility
	if visibility != VisibilityPublic {
		visibility = VisibilityPrivate
	}

	now := s.clock().UTC()
	id, err := randomHex(12)
	if err != nil {
		return Room{}, err
	}
	code, err := s.uniqueCode()
	if err != nil {
		return Room{}, err
	}

	room := Room{
		ID:         "room_" + id,
		Code:       code,
		GameID:     gameID,
		Visibility: visibility,
		Status:     StatusLobby,
		MaxPlayers: maxPlayers,
		HostUserID: host.ID,
		Options: Options{
			TurnSeconds:     60,
			MaxWaitSeconds:  180,
			AutoStart:       false,
			AllowSpectators: true,
		},
		RuleVotes:    []RuleVote{},
		GameRules:    map[string]any{},
		RuleMessages: []string{},
		Participants: []Participant{{
			User:      host,
			Ready:     false,
			Host:      true,
			SeatIndex: 0,
			JoinedAt:  now,
		}},
		Spectators: []Spectator{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.store.Save(room); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s *Service) List(filter ListFilter) []Room {
	rooms := s.store.List()
	result := make([]Room, 0, len(rooms))
	for _, room := range rooms {
		if !filter.IncludeClosed && room.Status == StatusClosed {
			continue
		}
		if filter.GameID != "" && room.GameID != filter.GameID {
			continue
		}
		if filter.Visibility != "" && room.Visibility != filter.Visibility {
			continue
		}
		result = append(result, room)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result
}

func (s *Service) JoinSpectator(roomID string, user guest.PublicUser) (Room, error) {
	user = normalizePublicUser(user, "spectator")
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	if room.HasParticipant(user.ID) || room.HasSpectator(user.ID) {
		return room, nil
	}
	if !room.Options.AllowSpectators {
		return Room{}, ErrSpectatorClosed
	}

	room.Spectators = append(room.Spectators, Spectator{User: user, JoinedAt: s.clock().UTC()})
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) UpdateOptions(roomID string, options Options) (Room, error) {
	return s.UpdateRoomSettings(roomID, UpdateOptionsRequest{Options: options})
}

func (s *Service) UpdateRoomSettings(roomID string, settings UpdateOptionsRequest) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	options := settings.Options
	if options.TurnSeconds < 10 {
		options.TurnSeconds = 10
	}
	if options.TurnSeconds > 300 {
		options.TurnSeconds = 300
	}
	if options.MaxWaitSeconds < 30 {
		options.MaxWaitSeconds = 30
	}
	if options.MaxWaitSeconds > 1800 {
		options.MaxWaitSeconds = 1800
	}
	room.Options.TurnSeconds = options.TurnSeconds
	room.Options.MaxWaitSeconds = options.MaxWaitSeconds
	room.Options.AutoStart = options.AutoStart
	room.Options.AllowSpectators = options.AllowSpectators
	if settings.MaxPlayers > 0 {
		room.MaxPlayers = clampInt(settings.MaxPlayers, len(room.Participants), 12)
	}
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) JoinByCode(code string, user guest.PublicUser) (Room, error) {
	room, err := s.store.FindByCode(normalizeCode(code))
	if err != nil {
		return Room{}, err
	}
	return s.joinRoomAsPlayerOrSpectator(room, user)
}

func (s *Service) JoinPublicRoom(roomID string, user guest.PublicUser) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	if room.Visibility != VisibilityPublic {
		return Room{}, ErrRoomNotJoinable
	}
	return s.joinRoomAsPlayerOrSpectator(room, user)
}

func (s *Service) joinRoomAsPlayerOrSpectator(room Room, user guest.PublicUser) (Room, error) {
	user = normalizePublicUser(user, "player")
	if room.Status == StatusPlaying {
		return s.JoinSpectator(room.ID, user)
	}
	if room.Status != StatusLobby {
		return Room{}, ErrRoomNotJoinable
	}
	if room.HasParticipant(user.ID) {
		return room, nil
	}
	if room.IsFull() {
		return s.JoinSpectator(room.ID, user)
	}

	now := s.clock().UTC()
	room.Participants = append(room.Participants, Participant{
		User:      user,
		Ready:     false,
		Host:      false,
		SeatIndex: len(room.Participants),
		JoinedAt:  now,
	})
	room.UpdatedAt = now

	if err := s.store.Save(room); err != nil {
		return Room{}, err
	}
	return room, nil
}

func normalizePublicUser(user guest.PublicUser, fallback string) guest.PublicUser {
	user.Nickname = guest.DisplayNickname(user.Nickname, user.ID)
	if user.ID == "" {
		user.ID = fallback
		user.Nickname = guest.DisplayNickname(user.Nickname, fallback)
	}
	return user
}

func (s *Service) FindByID(id string) (Room, error) {
	return s.store.FindByID(id)
}

func (s *Service) ToggleReady(roomID string, userID string, ready bool) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	for index := range room.Participants {
		if room.Participants[index].User.ID == userID {
			room.Participants[index].Ready = ready
			room.UpdatedAt = s.clock().UTC()
			return room, s.store.Save(room)
		}
	}
	return Room{}, ErrRoomNotFound
}

func (s *Service) SetStatus(roomID string, status Status) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	room.Status = status
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) SetPlaying(roomID string, sessionID string, playerIDs []string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	room.Status = StatusPlaying
	room.ActiveSessionID = sessionID
	room.PlayingPlayerIDs = append([]string(nil), playerIDs...)
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) ReturnToLobby(roomID string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	room.Status = StatusLobby
	room.ActiveSessionID = ""
	room.PlayingPlayerIDs = nil
	room.RuleMessages = nil
	for index := range room.Participants {
		room.Participants[index].Ready = false
	}
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) VoteRules(roomID string, userID string, choices map[string]string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	if room.Status != StatusLobby || !room.HasParticipant(userID) {
		return Room{}, ErrRoomNotJoinable
	}

	cleanChoices := make(map[string]string, len(choices))
	for key, value := range choices {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		cleanChoices[key] = value
	}
	if len(cleanChoices) == 0 {
		return Room{}, ErrRoomNotJoinable
	}

	now := s.clock().UTC()
	replaced := false
	for index := range room.RuleVotes {
		if room.RuleVotes[index].UserID == userID {
			room.RuleVotes[index].Choices = cleanChoices
			room.RuleVotes[index].CreatedAt = now
			replaced = true
			break
		}
	}
	if !replaced {
		room.RuleVotes = append(room.RuleVotes, RuleVote{UserID: userID, Choices: cleanChoices, CreatedAt: now})
	}
	room.GameRules = nil
	room.RuleMessages = nil
	for index := range room.Participants {
		room.Participants[index].Ready = false
	}
	room.UpdatedAt = now
	return room, s.store.Save(room)
}

func (s *Service) SetRuleResolution(roomID string, rules map[string]any, messages []string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	room.GameRules = cloneRuleMap(rules)
	room.RuleMessages = append([]string(nil), messages...)
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) Leave(roomID string, userID string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	participants := make([]Participant, 0, len(room.Participants))
	for _, participant := range room.Participants {
		if participant.User.ID != userID {
			participants = append(participants, participant)
		}
	}
	spectators := make([]Spectator, 0, len(room.Spectators))
	for _, spectator := range room.Spectators {
		if spectator.User.ID != userID {
			spectators = append(spectators, spectator)
		}
	}

	room.Participants = participants
	room.Spectators = spectators
	if len(room.Participants) == 0 {
		room.Status = StatusClosed
		room.HostUserID = ""
		room.ActiveSessionID = ""
	} else if room.HostUserID == userID {
		room.HostUserID = room.Participants[0].User.ID
		room.Participants[0].Host = true
	}

	for index := range room.Participants {
		room.Participants[index].SeatIndex = index
		room.Participants[index].Host = room.Participants[index].User.ID == room.HostUserID
	}

	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) Kick(roomID string, targetUserID string) (Room, error) {
	return s.Leave(roomID, targetUserID)
}

func (s *Service) TransferHost(roomID string, nextHostUserID string) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}

	if !room.HasParticipant(nextHostUserID) {
		return Room{}, ErrRoomNotFound
	}

	room.HostUserID = nextHostUserID
	for index := range room.Participants {
		room.Participants[index].Host = room.Participants[index].User.ID == nextHostUserID
	}
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func cloneRuleMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (s *Service) uniqueCode() (string, error) {
	for range 20 {
		code, err := randomCode(6)
		if err != nil {
			return "", err
		}
		if !s.store.CodeExists(code) {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique room code")
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func randomCode(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	builder.Grow(length)
	for range length {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		builder.WriteByte(alphabet[index.Int64()])
	}
	return builder.String(), nil
}

func randomHex(byteLength int) (string, error) {
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
