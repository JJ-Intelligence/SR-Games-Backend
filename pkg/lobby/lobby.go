package lobby

import (
	"sort"
	"sync"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
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
	Log *zap.Logger
	// Host is the host's player ID
	Host string

	// State is the state of the current game
	State game.GameState

	// PlayerIDToConnStore stores a mapping of Player IDs to Socket connections
	PlayerIDToConnStore map[string]*comms.ConnectionWrapper

	// RequestChannel stores a channel of incoming Requests
	RequestChannel chan comms.Request
}

func (l *Lobby) Close() {
	l.broadcastMessage(LobbyClosedBroadcast{})
}

func (l *Lobby) LobbyRequestHandler() {
	for {
		req := <-l.RequestChannel

		switch req.Message.Type {
		case "PlayerJoinedEvent", "PlayerLeftEvent":
			// New player joins the lobby
			players := make([]string, len(l.PlayerIDToConnStore))
			i := 0
			for player := range l.PlayerIDToConnStore {
				players[i] = player
				i++
			}
			sort.Strings(players)

			l.broadcastMessage(LobbyPlayerListBroadcast{
				PlayerIDs: players,
			})
		case "LobbyStartGameRequest":
			// Host starts a Game
			var contents LobbyStartGameRequest
			err := mapstructure.Decode(req.Message.Contents, &contents)
			if err != nil {
				req.ConnChannel <- comms.ToMessage(comms.ErrorResponse{
					Reason: "Unable to parse LobbyStartGameRequest %s",
					Error:  err,
				})
				continue
			}

			if req.PlayerID == l.Host {
				state, err := game.NewGameState(contents.Game, len(l.PlayerIDToConnStore))
				if err == nil {
					req.ConnChannel <- comms.ToMessage(LobbyStartGameStatusResponse{
						Status: false,
						// TODO: Return error message from NewGameState
						Reason: "Too many players",
					})
				} else {
					l.State = state
					req.ConnChannel <- comms.ToMessage(LobbyStartGameStatusResponse{
						Status: true,
					})
					l.broadcastMessage(LobbyStartGameBroadcast{Game: state.Name})
				}
			} else {
				req.ConnChannel <- comms.ToMessage(comms.ErrorResponse{
					Reason: "Only the host can start a game",
				})
			}
		}
	}
}

func (l Lobby) broadcastMessage(contents interface{}) {
	for _, conn := range l.PlayerIDToConnStore {
		conn.WriteChannel <- comms.ToMessage(contents)
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
