package room

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"board-game-platform/apps/api/internal/guest"
)

var (
	ErrRoomFull        = errors.New("room is full")
	ErrRoomNotJoinable = errors.New("room is not joinable")
)

type Clock func() time.Time

type Service struct {
	store Store
	clock Clock
}

func NewService(store Store, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: store, clock: clock}
}

func (s *Service) Create(host guest.PublicUser, gameID string, maxPlayers int) (Room, error) {
	if gameID == "" {
		gameID = "davinci"
	}
	if maxPlayers <= 0 {
		maxPlayers = 4
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
		Status:     StatusLobby,
		MaxPlayers: maxPlayers,
		HostUserID: host.ID,
		Options: Options{
			TurnSeconds: 60,
		},
		Participants: []Participant{{
			User:      host,
			Ready:     false,
			Host:      true,
			SeatIndex: 0,
			JoinedAt:  now,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Save(room); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s *Service) JoinSpectator(roomID string, user guest.PublicUser) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	if room.HasParticipant(user.ID) || room.HasSpectator(user.ID) {
		return room, nil
	}

	room.Spectators = append(room.Spectators, Spectator{User: user, JoinedAt: s.clock().UTC()})
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) UpdateOptions(roomID string, turnSeconds int) (Room, error) {
	room, err := s.store.FindByID(roomID)
	if err != nil {
		return Room{}, err
	}
	if turnSeconds < 10 {
		turnSeconds = 10
	}
	if turnSeconds > 300 {
		turnSeconds = 300
	}
	room.Options.TurnSeconds = turnSeconds
	room.UpdatedAt = s.clock().UTC()
	return room, s.store.Save(room)
}

func (s *Service) JoinByCode(code string, user guest.PublicUser) (Room, error) {
	room, err := s.store.FindByCode(normalizeCode(code))
	if err != nil {
		return Room{}, err
	}
	if room.Status != StatusLobby {
		return Room{}, ErrRoomNotJoinable
	}
	if room.HasParticipant(user.ID) {
		return room, nil
	}
	if room.IsFull() {
		return Room{}, ErrRoomFull
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
	for index := range room.Participants {
		room.Participants[index].Ready = false
	}
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
