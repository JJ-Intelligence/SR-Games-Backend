package comms

import (
	"github.com/gorilla/websocket"
)

// Request holds a Message and connection of a connected client.
type Request struct {
	ConnChannel chan Message
	PlayerID    string
	Message     Message
}

// ConnectionWrapper wraps a client connection, handling communication.
type ConnectionWrapper struct {
	Socket       *websocket.Conn
	WriteChannel chan Message
	PlayerID     string
}

func (c *ConnectionWrapper) ReadMessage() (Message, error) {
	var message Message
	err := c.Socket.ReadJSON(&message)
	return message, err
}

func (c *ConnectionWrapper) WriteMessage(message Message) error {
	return c.Socket.WriteJSON(message)
}

func (c *ConnectionWrapper) Close() {
	c.WriteChannel <- Message{Type: "CloseConnectionRequest"}
	c.Socket.Close()
}
