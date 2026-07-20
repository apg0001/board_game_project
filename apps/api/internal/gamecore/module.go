package gamecore

import (
	"context"
	"time"
)

type PlayerID string
type SessionID string
type GameID string

type Player struct {
	ID          PlayerID `json:"id"`
	SeatIndex   int      `json:"seatIndex"`
	DisplayName string   `json:"displayName"`
	Connected   bool     `json:"connected"`
}

type Context struct {
	SessionID  SessionID
	GameID     GameID
	Players    []Player
	Options    map[string]any
	Now        time.Time
	RandomSeed string
}

type Action struct {
	Type            string    `json:"type"`
	PlayerID        PlayerID  `json:"playerId"`
	Payload         any       `json:"payload"`
	ClientRequestID string    `json:"clientRequestId"`
	CreatedAt       time.Time `json:"createdAt"`
}

type Event struct {
	Type            string     `json:"type"`
	Visibility      Visibility `json:"visibility"`
	TargetPlayerIDs []PlayerID `json:"targetPlayerIds,omitempty"`
	Payload         any        `json:"payload"`
	Sequence        int64      `json:"sequence"`
}

type Visibility string

const (
	VisibilityPublic    Visibility = "PUBLIC"
	VisibilityPrivate   Visibility = "PRIVATE"
	VisibilitySpectator Visibility = "SPECTATOR"
)

type Result struct {
	PlayerID PlayerID `json:"playerId"`
	Rank     int      `json:"rank"`
	Score    int      `json:"score"`
	Outcome  Outcome  `json:"outcome"`
}

type Outcome string

const (
	OutcomeWin  Outcome = "WIN"
	OutcomeLose Outcome = "LOSE"
	OutcomeDraw Outcome = "DRAW"
)

type ActionResult struct {
	State  any
	Events []Event
}

type Module interface {
	ID() GameID
	Name() string
	MinPlayers() int
	MaxPlayers() int
	CreateInitialState(ctx Context) any
	PublicState(state any, viewerID PlayerID) any
	ValidateAction(ctx context.Context, state any, action Action, gameCtx Context) error
	ApplyAction(ctx context.Context, state any, action Action, gameCtx Context) (ActionResult, error)
	IsFinished(state any, gameCtx Context) bool
	CalculateResult(state any, gameCtx Context) []Result
}

type Registry struct {
	modules map[GameID]Module
}

func NewRegistry(modules ...Module) *Registry {
	registry := &Registry{modules: make(map[GameID]Module)}
	for _, module := range modules {
		registry.modules[module.ID()] = module
	}
	return registry
}

func (r *Registry) Find(id GameID) (Module, bool) {
	module, ok := r.modules[id]
	return module, ok
}
