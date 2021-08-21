package server

import "sync"

type Player struct {
	PlayerID string
	LobbyID  string
}

type Lobby struct {
	// Host is the host's player ID
	Host string

	// State is the state of the current game
	State interface{} // TODO: Set the GameState

	// PlayerIDToConnStore stores a mapping of Player IDs to Socket connections
	PlayerIDToConnStore map[string]ConnectionWrapper

	// RequestChannel stores a channel of incoming Requests
	RequestChannel chan Request
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
