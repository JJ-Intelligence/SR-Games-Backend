package config

import (
	"fmt"
	"io/ioutil"
	"plugin"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/game"
	"gopkg.in/yaml.v2"
)

type RawYamlConfig struct {
	Games map[string]string `yaml:"games"`
}

type Config struct {
	Games map[string]game.Game
}

func ParseConfig(path string) *Config {
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		panic("Unable to read config")
	}

	var rawConfig RawYamlConfig
	err = yaml.Unmarshal(configFile, &rawConfig)
	if err != nil {
		panic("Unable to parse yaml config")
	}

	games := make(map[string]game.Game)
	for name, pluginPath := range rawConfig.Games {
		p, err := plugin.Open(pluginPath)
		if err != nil {
			panic(fmt.Sprintf(
				"Unable to load game plugin from '%s': %s", pluginPath, err.Error()))
		}
		games[name] = game.NewGame(name, p)
	}
	return &Config{Games: games}
}
