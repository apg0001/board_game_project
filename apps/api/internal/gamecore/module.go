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

type ActionResult[TState any] struct {
	State  TState
	Events []Event
}

type Module[TState any] interface {
	ID() GameID
	Name() string
	MinPlayers() int
	MaxPlayers() int
	CreateInitialState(ctx Context) TState
	PublicState(state TState, viewerID PlayerID) any
	ValidateAction(ctx context.Context, state TState, action Action, gameCtx Context) error
	ApplyAction(ctx context.Context, state TState, action Action, gameCtx Context) (ActionResult[TState], error)
	IsFinished(state TState, gameCtx Context) bool
	CalculateResult(state TState, gameCtx Context) []Result
}
