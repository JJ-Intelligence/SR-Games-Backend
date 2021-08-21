package server

import "github.com/gorilla/websocket"

// Request holds a Message and connection of a connected client.
type Request struct {
	ConnChannel chan Message
	Data        []byte
}

// ConnectionWrapper wraps a client connection, handling communication.
type ConnectionWrapper struct {
	socket       *websocket.Conn
	WriteChannel chan Message
}

func (c *ConnectionWrapper) ReadMessage() ([]byte, error) {
	_, bytes, err := c.socket.ReadMessage()
	return bytes, err
}

func (c *ConnectionWrapper) WriteMessage(message Message) error {
	return c.socket.WriteJSON(message)
}

func (c *ConnectionWrapper) Close() {
	c.WriteChannel <- Message{Type: "CloseConnectionRequest"}
	c.socket.Close()
}
