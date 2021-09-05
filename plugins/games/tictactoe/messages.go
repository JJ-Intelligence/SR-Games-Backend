package main

type PlayerSymbolsBroadcast struct {
	PlayerNought string `json:"playerNought"`
	PlayerCross  string `json:"playerCross"`
}

type PlayerTurnBroadcast struct {
	PlayerID string `json:"playerID"`
}

type MakeMoveRequest struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MakeMoveResponse struct {
	Status bool `json:"status"`
}

type MakeMoveBroadcast struct {
	X        int    `json:"x"`
	Y        int    `json:"y"`
	PlayerID string `json:"playerID"`
}

type WinnerBroadcast struct {
	PlayerID string `json:"playerID"`
}
