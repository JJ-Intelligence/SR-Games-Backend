package server

import (
	"github.com/gorilla/websocket"
	"log"
)

// Request holds a Message and connection of a connected client.
type Request struct {
	Socket  *websocket.Conn
	Message *Message
}

// Message represents JSON data sent across socket connections.
type Message struct {
	Type     string `json:"type"`
	Code     string `json:"code,omitempty"`
	Contents string `json:"message,omitempty"`
}

// SendMessage writes the Message as a JSON to the given socket.
func SendMessage(message *Message, socket *websocket.Conn) {
	err := socket.WriteJSON(message)
	if err != nil {
		log.Printf("ERROR sending message of type '%s' - %s", message.Type, err)
	}
}
