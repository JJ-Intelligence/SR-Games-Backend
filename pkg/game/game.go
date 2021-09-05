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

type GameService struct {
	NewState      func([]string) (interface{}, error)
	HandleRequest func(chan GameRequest, interface{}, string, string, interface{}) interface{}
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
		NewState:      newState.(func([]string) (interface{}, error)),
		HandleRequest: handleRequest.(func(chan GameRequest, interface{}, string, string, interface{}) interface{}),
	}
}
