package config

import (
	"fmt"
	"io/ioutil"
	"plugin"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
	"gopkg.in/yaml.v2"
)

type RawYamlConfig struct {
	Games map[string]string `yaml:"games"`
}

type Config struct {
	Games map[string]Game
}

type GameRequest struct {
	Players []string
	Message comms.Message
}

type newStateFunc func(players []string) (interface{}, error)
type handleRequestFunc func(
	gameChan chan GameRequest, state interface{},
	player, messageType string, contents interface{})

type Game struct {
	NewState      newStateFunc
	HandleRequest handleRequestFunc
}

func NewGame(name string, p *plugin.Plugin) Game {
	newState, err := p.Lookup("NewState")
	if err != nil {
		panic(fmt.Sprintf("NewState function does not exist for plugin %s", name))
	}
	handleRequest, err := p.Lookup("HandleRequest")
	if err != nil {
		panic(fmt.Sprintf("HandleRequest function does not exist for plugin %s", name))
	}

	return Game{
		NewState:      newState.(newStateFunc),
		HandleRequest: handleRequest.(handleRequestFunc),
	}
}

func ParseConfig(path string) *Config {
	configFile, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		panic("Unable to read config")
	}

	var rawConfig RawYamlConfig
	err = yaml.Unmarshal(configFile, rawConfig)
	if err != nil {
		panic("Unable to parse yaml config")
	}

	games := make(map[string]Game)
	for name, pluginPath := range rawConfig.Games {
		p, err := plugin.Open(pluginPath)
		if err != nil {
			panic(fmt.Sprintf("Unable to load game plugin: %e", err))
		}
		games[name] = NewGame(name, p)
	}
	return &Config{Games: games}
}
