package server

import (
	"github.com/gorilla/websocket"
	"hash/fnv"
	"math/rand"
	"strconv"
	"time"
)

// Request holds a Message and connection of a connected client.
type Request struct {
	Connection ConnectionWrapper
	Message    Message
}

// Message represents JSON data sent across socket connections.
type Message struct {
	Type     string `json:"type"`
	Code     string `json:"code,omitempty"`
	Contents string `json:"message,omitempty"`
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
	code   string
}

func (s *SocketWrapper) ReadMessage() (Message, error) {
	var message Message
	err := s.socket.ReadJSON(&message)
	return message, err
}

func (s *SocketWrapper) WriteMessage(message Message) error {
	return s.socket.WriteJSON(message)
}

type ConnectionStore interface {
	// NewCode generates a new unique code.
	NewCode() string

	Connect(code string, conn ConnectionWrapper)
	Disconnect(conn ConnectionWrapper)
	GetConnectionsByCode(code string) []ConnectionWrapper
}

type MapConnectionStore struct {
	codeStore map[string]map[ConnectionWrapper]bool
	connStore map[ConnectionWrapper]string
}

// NewCode returns a hashed random int, seeded with the current time.
func (m *MapConnectionStore) NewCode() string {
	rand.Seed(time.Now().UnixNano())
	num := rand.Int()
	h := fnv.New32()
	h.Write([]byte(strconv.Itoa(num)))
	return strconv.Itoa(int(h.Sum32()))
}

func (m *MapConnectionStore) Connect(code string, conn ConnectionWrapper) {
	m.connStore[conn] = code
	if _, ok := m.codeStore[code]; ok {
		m.codeStore[code][conn] = true
	} else {
		m.codeStore[code] = map[ConnectionWrapper]bool{
			conn: true,
		}
	}
}

func (m *MapConnectionStore) Disconnect(conn ConnectionWrapper) {
	if code, ok := m.connStore[conn]; ok {
		delete(m.connStore, conn)
		if len(m.codeStore[code]) == 1 {
			delete(m.codeStore, code)
		} else {
			delete(m.codeStore[code], conn)
		}
	}
}

func (m *MapConnectionStore) GetConnectionsByCode(code string) []ConnectionWrapper {
	if val, ok := m.codeStore[code]; ok {
		conns := make([]ConnectionWrapper, 0, len(val))
		for k := range val {
			conns = append(conns, k)
		}
		return conns
	}

	return nil
}
