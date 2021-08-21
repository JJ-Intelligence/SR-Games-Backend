package server

// Message
type Message struct {
	Type     string      `json:"type"`
	Contents interface{} `json:"contents"`
}

func (m *Message) UnmarshalJSON(data []byte) error {

}

type LobbyJoinRequest struct {
	PlayerID string `json:"playerID"`
	LobbyID  string `json:"lobbyID"`
}
