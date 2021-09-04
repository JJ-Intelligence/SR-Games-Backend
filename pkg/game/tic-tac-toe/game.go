package tictactoe

import (
	"fmt"
)

const NUM_PLAYERS = 2

type State struct {
	Players []string // {Nought, Cross}

	Board [3][3]rune
}

func NewState(players []string) (*State, error) {
	if len(players) != NUM_PLAYERS {
		return nil, fmt.Errorf("invalid number of players, should be %d", NUM_PLAYERS)
	}

	return &State{
		Players: players,
		Board:   [3][3]rune{},
	}, nil
}
