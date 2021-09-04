package lobby

// Lobby Player management
type LobbyJoinRequest struct {
	PlayerID string `json:"playerID"`
	LobbyID  string `json:"lobbyID"`
}

type PlayerJoinedEvent struct{}

type LobbyLeaveRequest struct{}

type PlayerLeftEvent struct{}

type LobbyPlayerListBroadcast struct {
	PlayerIDs []string `json:"playerIDs"`
}

// Starting a Game
type LobbyStartGameRequest struct {
	Game string `json:"game"`
}

type LobbyStartGameResponse struct {
	Status bool   `json:"status"`
	Reason string `json:"reason"`
}

type LobbyStartGameBroadcast struct {
	Game string `json:"game"`
}

type LobbyClosedBroadcast struct{}

type LobbyDoesNotExistResponse struct{}
