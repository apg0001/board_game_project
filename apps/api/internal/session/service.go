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
	return s.StartWithPlayers(room, nil)
}

func (s *Service) StartWithPlayers(room room.Room, playerIDs []string) (Session, error) {
	module, ok := s.registry.Find(gamecore.GameID(room.GameID))
	if !ok {
		return Session{}, ErrGameNotRegistered
	}
	startRoom := room
	if len(playerIDs) > 0 {
		var ok bool
		startRoom, ok = roomWithSelectedPlayers(room, playerIDs)
		if !ok {
			return Session{}, ErrRoomNotReady
		}
	}
	if len(startRoom.Participants) > module.MaxPlayers() {
		return Session{}, ErrRoomOverCapacity
	}
	if !canStart(startRoom, module) {
		return Session{}, ErrRoomNotReady
	}

	now := s.clock().UTC()
	id, err := randomID()
	if err != nil {
		return Session{}, err
	}
	gameCtx := s.contextFromRoom(startRoom, now)
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

func roomWithSelectedPlayers(source room.Room, playerIDs []string) (room.Room, bool) {
	seen := map[string]struct{}{}
	selected := make([]room.Participant, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		if _, exists := seen[playerID]; exists {
			return room.Room{}, false
		}
		seen[playerID] = struct{}{}
		for _, participant := range source.Participants {
			if participant.User.ID == playerID {
				selected = append(selected, participant)
				break
			}
		}
	}
	if len(selected) != len(playerIDs) {
		return room.Room{}, false
	}
	for index := range selected {
		selected[index].SeatIndex = index
	}
	source.Participants = selected
	return source, true
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

	gameCtx := s.contextFromRoom(room, s.clock().UTC())
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

	gameCtx := s.contextFromRoom(room, s.clock().UTC())
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

func (s *Service) ResolveRuleOptions(room room.Room) gamecore.RuleResolution {
	module, ok := s.registry.Find(gamecore.GameID(room.GameID))
	if !ok {
		return gamecore.RuleResolution{Options: cloneOptions(room.GameRules)}
	}
	resolver, ok := module.(gamecore.RuleResolver)
	if !ok {
		return gamecore.RuleResolution{Options: cloneOptions(room.GameRules)}
	}

	votes := make([]gamecore.RuleVote, 0, len(room.RuleVotes))
	for _, vote := range room.RuleVotes {
		votes = append(votes, gamecore.RuleVote{
			UserID:  vote.UserID,
			Choices: cloneStringMap(vote.Choices),
		})
	}
	resolution := resolver.ResolveRules(votes, room.ID)
	if resolution.Options == nil {
		resolution.Options = map[string]any{}
	}
	return resolution
}

func (s *Service) contextFromRoom(room room.Room, now time.Time) gamecore.Context {
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
		Options:    cloneOptions(room.GameRules),
		Now:        now,
		RandomSeed: room.ID,
	}
}

func cloneOptions(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func randomID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
