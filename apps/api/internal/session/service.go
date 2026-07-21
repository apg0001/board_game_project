package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/room"
)

var (
	ErrGameNotRegistered  = errors.New("game module is not registered")
	ErrRoomNotReady       = errors.New("room is not ready to start")
	ErrRoomOverCapacity   = errors.New("room has too many players for selected game")
	ErrTimeoutUnsupported = errors.New("game module does not support timeout handling")
)

type Clock func() time.Time

type Service struct {
	store    Store
	registry *gamecore.Registry
	clock    Clock
}

func NewService(store Store, registry *gamecore.Registry, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{store: store, registry: registry, clock: clock}
}

func (s *Service) Start(room room.Room) (Session, error) {
	module, ok := s.registry.Find(gamecore.GameID(room.GameID))
	if !ok {
		return Session{}, ErrGameNotRegistered
	}
	if len(room.Participants) > module.MaxPlayers() {
		return Session{}, ErrRoomOverCapacity
	}
	if !canStart(room, module) {
		return Session{}, ErrRoomNotReady
	}

	now := s.clock().UTC()
	id, err := randomID()
	if err != nil {
		return Session{}, err
	}
	gameCtx := contextFromRoom(room, now)
	created := Session{
		ID:        "session_" + id,
		RoomID:    room.ID,
		GameID:    room.GameID,
		Status:    StatusActive,
		State:     module.CreateInitialState(gameCtx),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Save(created); err != nil {
		return Session{}, err
	}
	return created, nil
}

func (s *Service) FindByID(id string) (Session, error) {
	return s.store.FindByID(id)
}

func (s *Service) PublicView(room room.Room, session Session, viewerID gamecore.PlayerID) (Session, error) {
	module, ok := s.registry.Find(gamecore.GameID(session.GameID))
	if !ok {
		return Session{}, ErrGameNotRegistered
	}

	view := session
	view.State = module.PublicState(session.State, viewerID)
	return view, nil
}

func (s *Service) ApplyAction(ctx context.Context, room room.Room, sessionID string, action gamecore.Action) (Session, []gamecore.Event, error) {
	current, err := s.store.FindByID(sessionID)
	if err != nil {
		return Session{}, nil, err
	}

	module, ok := s.registry.Find(gamecore.GameID(current.GameID))
	if !ok {
		return Session{}, nil, ErrGameNotRegistered
	}

	gameCtx := contextFromRoom(room, s.clock().UTC())
	if err := module.ValidateAction(ctx, current.State, action, gameCtx); err != nil {
		return Session{}, nil, err
	}

	result, err := module.ApplyAction(ctx, current.State, action, gameCtx)
	if err != nil {
		return Session{}, nil, err
	}

	current.State = result.State
	current.UpdatedAt = s.clock().UTC()
	if module.IsFinished(current.State, gameCtx) {
		current.Status = StatusFinished
		current.Results = module.CalculateResult(current.State, gameCtx)
	}

	if err := s.store.Save(current); err != nil {
		return Session{}, nil, err
	}
	return current, result.Events, nil
}

func (s *Service) ApplyTimeout(ctx context.Context, room room.Room, sessionID string, playerID gamecore.PlayerID) (Session, []gamecore.Event, error) {
	current, err := s.store.FindByID(sessionID)
	if err != nil {
		return Session{}, nil, err
	}
	if current.Status != StatusActive {
		return current, nil, nil
	}

	module, ok := s.registry.Find(gamecore.GameID(current.GameID))
	if !ok {
		return Session{}, nil, ErrGameNotRegistered
	}
	timeoutHandler, ok := module.(gamecore.TimeoutHandler)
	if !ok {
		return Session{}, nil, ErrTimeoutUnsupported
	}

	gameCtx := contextFromRoom(room, s.clock().UTC())
	result, err := timeoutHandler.ApplyTimeout(ctx, current.State, playerID, gameCtx)
	if err != nil {
		return Session{}, nil, err
	}

	current.State = result.State
	current.UpdatedAt = s.clock().UTC()
	if module.IsFinished(current.State, gameCtx) {
		current.Status = StatusFinished
		current.Results = module.CalculateResult(current.State, gameCtx)
	}

	if err := s.store.Save(current); err != nil {
		return Session{}, nil, err
	}
	return current, result.Events, nil
}

func canStart(room room.Room, module gamecore.Module) bool {
	if len(room.Participants) < module.MinPlayers() || len(room.Participants) > module.MaxPlayers() {
		return false
	}
	for _, participant := range room.Participants {
		if !participant.Ready {
			return false
		}
	}
	return true
}

func contextFromRoom(room room.Room, now time.Time) gamecore.Context {
	players := make([]gamecore.Player, 0, len(room.Participants))
	for _, participant := range room.Participants {
		players = append(players, gamecore.Player{
			ID:          gamecore.PlayerID(participant.User.ID),
			SeatIndex:   participant.SeatIndex,
			DisplayName: participant.User.Nickname,
			Connected:   true,
		})
	}
	return gamecore.Context{
		SessionID:  "",
		GameID:     gamecore.GameID(room.GameID),
		Players:    players,
		Options:    map[string]any{},
		Now:        now,
		RandomSeed: room.ID,
	}
}

func randomID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
