package main

import (
	"fmt"
	"math/rand"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/game"
	"github.com/mitchellh/mapstructure"
)

const NUM_PLAYERS = 2

type State struct {
	Players []string // {Nought, Cross}
	Board   [3][3]int

	currentPlayer int
}

func (s State) isValidMove(x, y int) bool {
	return x >= 0 && y >= 0 && x < len(s.Board) && y < len(s.Board) && s.Board[x][y] == 0
}

func (s State) isWinner(x, y int) bool {
	player := s.Board[x][y]
	return (player == s.Board[x-1%len(s.Board)][y] && player == s.Board[x+1%len(s.Board)][y]) ||
		(player == s.Board[x][y-1%len(s.Board)] && player == s.Board[x][y+1%len(s.Board)]) ||
		((x == 0 && y == 0 || x == 1 && y == 1 || x == 2 && y == 2) &&
			player == s.Board[x-1%len(s.Board)][y-1%len(s.Board)] &&
			player == s.Board[x+1%len(s.Board)][y+1%len(s.Board)]) ||
		((x == 2 && y == 0 || x == 1 && y == 1 || x == 0 && y == 2) &&
			player == s.Board[x-1%len(s.Board)][y+1%len(s.Board)] &&
			player == s.Board[x+1%len(s.Board)][y-1%len(s.Board)])
}

func NewState(players []string) (interface{}, error) {
	if len(players) != NUM_PLAYERS {
		return nil, fmt.Errorf("invalid number of players, should be %d", NUM_PLAYERS)
	}

	return &State{
		Players: players,
		Board:   [3][3]int{},
	}, nil
}

func HandleRequest(
	gameChan chan game.GameRequest,
	stateInterface interface{},
	player,
	messageType string,
	messageContents interface{},
) interface{} {
	// Decode state
	state := stateInterface.(*State)

	// Handle the request
	switch messageType {
	case "StartGameRequest":
		// Start the game
		gameChan <- game.GameRequest{
			Players: state.Players,
			Message: comms.ToMessage(PlayerSymbolsBroadcast{
				PlayerNought: state.Players[0],
				PlayerCross:  state.Players[1],
			}),
		}
		state.currentPlayer = rand.Intn(len(state.Players))
		gameChan <- game.GameRequest{
			Players: state.Players,
			Message: comms.ToMessage(PlayerTurnBroadcast{
				PlayerID: state.Players[state.currentPlayer],
			}),
		}

	case "MakeMoveRequest":
		// A player makes a move
		currentPlayer := state.Players[state.currentPlayer]
		if player == currentPlayer {
			var contents MakeMoveRequest
			err := mapstructure.Decode(messageContents, &contents)

			if err == nil {
				if state.isValidMove(contents.X, contents.Y) {
					// Update state and inform players of move
					state.Board[contents.X][contents.Y] = state.currentPlayer + 1
					state.currentPlayer = (state.currentPlayer + 1) % len(state.Players)
					gameChan <- game.GameRequest{
						Players: []string{player},
						Message: comms.ToMessage(MakeMoveResponse{true}),
					}
					gameChan <- game.GameRequest{
						Players: state.Players,
						Message: comms.ToMessage(MakeMoveBroadcast{
							X:        contents.X,
							Y:        contents.Y,
							PlayerID: player,
						}),
					}

					if state.isWinner(contents.X, contents.Y) {
						// The current player has won the game
						gameChan <- game.GameRequest{
							Players: state.Players,
							Message: comms.ToMessage(WinnerBroadcast{player}),
						}
					} else {
						// Next player is making a move
						gameChan <- game.GameRequest{
							Players: state.Players,
							Message: comms.ToMessage(
								PlayerTurnBroadcast{state.Players[state.currentPlayer]}),
						}
					}
				} else {
					return MakeMoveResponse{false}
				}
			} else {
				return comms.ErrorDecodingMessageResponse{}
			}
		} else {
			return comms.ErrorResponse{Reason: "Not your turn"}
		}
	}

	return nil
}
