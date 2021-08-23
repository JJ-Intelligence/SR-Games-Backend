package comms

import (
	"reflect"
)

// Messages used in conversation with a client
type Message struct {
	Type     string      `json:"type"`
	Contents interface{} `json:"contents"`
}

// Convert message contents into a Message
func ToMessage(contents interface{}) Message {
	return Message{
		Type:     reflect.TypeOf(contents).Name(),
		Contents: contents,
	}
}

// Error returned to the client
type ErrorResponse struct {
	Reason string `json:"reason"`
	Error  error  `json:"error"`
}
