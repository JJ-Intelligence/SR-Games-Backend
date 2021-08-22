package game

type GameState struct {
	Name  string
	State interface{}
}

func NewGameState(name string, numPlayers int) (GameState, error) {
	// TODO: Check num players
	return GameState{Name: name, State: nil}, nil
}
