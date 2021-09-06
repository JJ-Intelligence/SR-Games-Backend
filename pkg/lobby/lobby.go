package lobby

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/config"
	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/game"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"

	"go.uber.org/zap"
)

type Player struct {
	PlayerID string
	LobbyID  string
}

func IsValidPlayerID(playerID string) bool {
	_, err := uuid.Parse(playerID)
	return err == nil
}

type Lobby struct {
	Log     *zap.Logger
	LobbyID string
	// Host is the host's player ID
	Host string

	// State of the current game
	GameName        string
	GameState       interface{}
	GameRequestChan chan game.GameRequest

	// PlayerIDToConnStore stores a mapping of Player IDs to Socket connections
	PlayerIDToConnStore map[string]*comms.ConnectionWrapper

	// RequestChannel stores a channel of incoming Requests
	RequestChannel chan comms.Request
}

func (l *Lobby) Close() {
	l.broadcastMessageToLobby(LobbyClosedBroadcast{})
	close(l.RequestChannel)
	if l.GameRequestChan != nil {
		close(l.GameRequestChan)
	}
}

func (l *Lobby) LobbyRequestHandler(config *config.Config) {
	for {
		req := <-l.RequestChannel

		switch req.Message.Type {
		case "PlayerJoinedEvent", "PlayerLeftEvent":
			// New player joins the lobby
			players := l.getPlayersList()
			sort.Strings(players)

			l.broadcastMessageToLobby(LobbyPlayerListBroadcast{
				PlayerIDs: players,
			})

		case "LobbyStartGameRequest":
			// Host starts a Game
			var contents LobbyStartGameRequest
			err := mapstructure.Decode(req.Message.Contents, &contents)
			if err != nil {
				req.Error("Unable to parse LobbyStartGameRequest", err)
				continue
			}

			if req.PlayerID == l.Host {
				if gamePlugin, ok := config.Games[contents.Game]; ok {
					state, err := gamePlugin.NewState(l.getPlayersList())
					if err == nil {
						// Save game state to Lobby
						l.GameName = contents.Game
						l.GameState = state
						l.GameRequestChan = make(chan game.GameRequest)

						// Run a handler to handle requests from the GameService
						go l.GameRequestHandler()

						// Tell players that the game has started
						req.ConnChannel <- comms.ToMessage(LobbyStartGameResponse{
							Status: true,
						})
						l.broadcastMessageToLobby(
							LobbyStartGameBroadcast{Game: l.GameName})
						l.Log.Info(fmt.Sprintf(
							"Started new game of %s in lobby %s", l.GameName, l.LobbyID))
					} else {
						req.ConnChannel <- comms.ToMessage(LobbyStartGameResponse{
							Status: false,
							Reason: err.Error(),
						})
					}
				} else {
					req.Error("Invalid game name", nil)
				}
			} else {
				req.Error(fmt.Sprintf(
					"Only the host can start a game (player %s, host %s)",
					req.PlayerID,
					l.Host,
				), nil)
			}

		default:
			// Route non-lobby-related messages
			typeComponents := strings.Split(req.Message.Type, "/")

			switch typeComponents[0] {
			case "Game":
				if len(typeComponents) != 2 {
					req.Error(fmt.Sprintf(
						"%s is an invalid Game message type, it should be of the format "+
							"'Game/<game-message-type>'",
						req.Message.Type,
					), nil)
				} else if l.GameState == nil {
					req.Error("Must set LobbyStartGameRequest first", nil)
				} else {
					errMessage := config.Games[l.GameName].HandleRequest(
						l.GameRequestChan, l.GameState, req.PlayerID,
						typeComponents[1], req.Message.Contents)
					if errMessage != nil {
						req.ConnChannel <- comms.ToMessage(errMessage)
					}
				}

			default:
				req.Error(
					fmt.Sprintf("%s is an invalid message type", req.Message.Type), nil)
			}
		}
	}
}

func (l *Lobby) broadcastMessageToLobby(contents interface{}) {
	for _, conn := range l.PlayerIDToConnStore {
		conn.WriteChannel <- comms.ToMessage(contents)
	}
}

func (l *Lobby) broadcastMessageToPlayers(message comms.Message, players []string) {
	for _, player := range players {
		l.PlayerIDToConnStore[player].WriteChannel <- message
	}
}

func (l *Lobby) getPlayersList() []string {
	players := make([]string, len(l.PlayerIDToConnStore))
	i := 0
	for player := range l.PlayerIDToConnStore {
		players[i] = player
		i++
	}
	return players
}

// Reads in requests from games and sends them to players
func (l *Lobby) GameRequestHandler() {
	for {
		req := <-l.GameRequestChan
		l.broadcastMessageToPlayers(
			comms.Message{
				Type:     "Game/" + req.Message.Type,
				Contents: req.Message.Contents,
			},
			req.Players,
		)
	}
}

// LobbyStoreMap stores Lobby IDs mapped to Lobby structs
type LobbyStore struct {
	// We're using a sync.Map which is optimised for few writes but lots of reads
	store sync.Map
}

func (s *LobbyStore) Put(key string, value *Lobby) {
	s.store.Store(key, value)
}

func (s *LobbyStore) Get(key string) (*Lobby, bool) {
	if value, ok := s.store.Load(key); ok {
		return value.(*Lobby), true
	}
	return nil, false
}

func (s *LobbyStore) Delete(key string) {
	s.store.Delete(key)
}
