package server

import "github.com/gorilla/websocket"

// Request holds a Message and connection of a connected client.
type Request struct {
	Connection ConnectionWrapper
	Message    Message
}

// ConnectionWrapper wraps a client connection, handling communication.
type ConnectionWrapper interface {
	// ReadMessage reads a message in from the client.
	ReadMessage() (Message, error)

	// WriteMessage sends a message to the client.
	WriteMessage(message Message) error
}

// SocketWrapper is a ConnectionWrapper which wraps a client websocket connection.
type SocketWrapper struct {
	socket *websocket.Conn
}

func (s *SocketWrapper) ReadMessage() ([]byte, error) {
	_, bytes, err := s.socket.ReadMessage()
	return message, err
}

func (s *SocketWrapper) WriteMessage(message Message) error {
	return s.socket.WriteJSON(message)
}
