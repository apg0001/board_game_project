package davinci

import (
	"context"
	"errors"

	"board-game-platform/apps/api/internal/gamecore"
)

type State struct {
	TurnIndex int      `json:"turnIndex"`
	Round     int      `json:"round"`
	Log       []string `json:"log"`
	Finished  bool     `json:"finished"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "davinci"
}

func (m Module) Name() string {
	return "다빈치 코드"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 4
}

func (m Module) CreateInitialState(_ gamecore.Context) any {
	return State{
		TurnIndex: 0,
		Round:     1,
		Log:       []string{"게임 세션이 시작되었습니다."},
		Finished:  false,
	}
}

func (m Module) PublicState(state any, _ gamecore.PlayerID) any {
	return asState(state)
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	if action.Type != "demo.advance" && action.Type != "demo.finish" {
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, gameCtx gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)

	switch action.Type {
	case "demo.advance":
		current.Log = append(current.Log, string(action.PlayerID)+" 님이 턴을 진행했습니다.")
		if len(gameCtx.Players) > 0 {
			current.TurnIndex = (current.TurnIndex + 1) % len(gameCtx.Players)
		}
		current.Round++
	case "demo.finish":
		current.Log = append(current.Log, string(action.PlayerID)+" 님이 게임 종료를 요청했습니다.")
		current.Finished = true
	}

	return gamecore.ActionResult{
		State: current,
		Events: []gamecore.Event{{
			Type:       "game.state_updated",
			Visibility: gamecore.VisibilityPublic,
			Payload:    current,
		}},
	}, nil
}

func (m Module) IsFinished(state any, _ gamecore.Context) bool {
	return asState(state).Finished
}

func (m Module) CalculateResult(_ any, gameCtx gamecore.Context) []gamecore.Result {
	results := make([]gamecore.Result, 0, len(gameCtx.Players))
	for index, player := range gameCtx.Players {
		outcome := gamecore.OutcomeLose
		if index == 0 {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: player.ID,
			Rank:     index + 1,
			Score:    len(gameCtx.Players) - index,
			Outcome:  outcome,
		})
	}
	return results
}

func asState(state any) State {
	typed, ok := state.(State)
	if ok {
		return typed
	}

	fromMap, ok := state.(map[string]any)
	if !ok {
		return State{}
	}

	result := State{}
	if value, ok := fromMap["turnIndex"].(float64); ok {
		result.TurnIndex = int(value)
	}
	if value, ok := fromMap["round"].(float64); ok {
		result.Round = int(value)
	}
	if value, ok := fromMap["finished"].(bool); ok {
		result.Finished = value
	}
	if values, ok := fromMap["log"].([]any); ok {
		for _, value := range values {
			if text, ok := value.(string); ok {
				result.Log = append(result.Log, text)
			}
		}
	}
	return result
}
