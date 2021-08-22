package comms

import (
	"encoding/json"
	"reflect"
)

// Messages used in conversation with a client
type Message struct {
	Type     string `json:"type"`
	Contents []byte `json:"contents"`
}

// Convert message contents into a Message
func ToMessage(contents interface{}) Message {
	jsonContents, _ := json.Marshal(contents)
	return Message{
		Type:     reflect.TypeOf(contents).Name(),
		Contents: jsonContents,
	}
}

// Error returned to the client
type ErrorResponse struct {
	Reason string `json:"reason"`
}
