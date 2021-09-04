package game

import (
	"fmt"

	tictactoe "github.com/JJ-Intelligence/SR-Games-Backend/pkg/game/tic-tac-toe"
)

type GameState struct {
	Name  string
	State interface{}
}

func NewGameState(name string, players []string) (*GameState, error) {
	var state interface{}
	var err error
	switch name {
	case "tictactoe":
		state, err = tictactoe.NewState(players)
	default:
		err = fmt.Errorf("invalid game name")
	}

	if err == nil {
		return &GameState{Name: name, State: state}, nil
	}
	return nil, err
}
