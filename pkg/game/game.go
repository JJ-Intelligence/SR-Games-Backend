package game

import (
	"fmt"
	"plugin"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
)

type GameRequest struct {
	Players []string
	Message comms.Message
}

type newStateFunc func(players []string) (state interface{}, err error)
type handleRequestFunc func(
	gameChan chan GameRequest, state interface{},
	player, messageType string, contents interface{})

type GameService struct {
	NewState      newStateFunc
	HandleRequest handleRequestFunc
}

func NewGame(name string, p *plugin.Plugin) GameService {
	newState, err := p.Lookup("NewState")
	if err != nil {
		panic(fmt.Sprintf("NewState function does not exist for plugin %s", name))
	}
	handleRequest, err := p.Lookup("HandleRequest")
	if err != nil {
		panic(fmt.Sprintf("HandleRequest function does not exist for plugin %s", name))
	}

	return GameService{
		NewState: newState.(func(players []string) (state interface{}, err error)),
		HandleRequest: handleRequest.(func(
			gameChan chan GameRequest, state interface{},
			player, messageType string, contents interface{})),
	}
}
