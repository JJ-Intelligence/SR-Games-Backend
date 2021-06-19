package server

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type BroadcastChannel chan Request

// Server stores all connection dependencies for the websocket server.
type Server struct {
	store          ConnectionStore
	broadcast      BroadcastChannel
	socketUpgrader websocket.Upgrader
}

// NewServer constructs a new Server instance.
func NewServer(checkOriginFunc func(r *http.Request) bool) *Server {
	return &Server{
		store:          &MapConnectionStore{},
		broadcast:      make(BroadcastChannel),
		socketUpgrader: websocket.Upgrader{CheckOrigin: checkOriginFunc},
	}
}

// Start starts up the websocket server.
func (server Server) Start(port string) {
	// Concurrently deliver client messages
	go broadcastHandler(server.store, server.broadcast)
	// Handle incoming requests
	http.HandleFunc("/", connectionHandler(server.store, server.broadcast, server.socketUpgrader))

	log.Println("Started server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ERROR server failed during ListenAndServer -", err)
	}
}

// connectionHandler upgrades new HTTP requests from clients to websockets, reading in further messages from
// those clients.
func connectionHandler(store ConnectionStore, broadcast BroadcastChannel, upgrader websocket.Upgrader) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP GET request to a socket connection
		s, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("ERROR upgrading connection -", err)
			return
		}
		socket := &SocketWrapper{socket: s}
		defer func() {
			store.Disconnect(socket)
			socket.socket.Close()
		}()

		// Forever handle messages from this new client
		for {
			err = handleIncomingMessage(socket, broadcast)
			if err != nil {
				log.Println("Client errored or disconnected", err)
				return
			}
		}
	}
}

// handleIncomingMessage reads messages from a socket and sends them to the broadcast channel, returning an error
// if the client has disconnected.
func handleIncomingMessage(conn ConnectionWrapper, broadcast BroadcastChannel) error {
	message, err := conn.ReadMessage()

	if websocket.IsUnexpectedCloseError(err) {
		log.Println("Client errored or disconnected", err)
		return err
	} else if err != nil {
		log.Println("ERROR reading incoming message -", err)
	} else {
		broadcast <- Request{Connection: conn, Message: message}
	}
	return nil
}

// broadcastHandler reads in messages from a MessageChannel and forwards them on or replies to clients.
func broadcastHandler(store ConnectionStore, broadcast BroadcastChannel) {
	for {
		// Pop the next message off the broadcast channel and send it
		request := <-broadcast
		switch request.Message.Type {

		case "Create":
			// Generate new room code and connect the new client
			code := store.NewCode()
			store.Connect(code, request.Connection)
			err := request.Connection.WriteMessage(Message{Type: "Create", Code: code})
			if err != nil {
				log.Println("ERROR sending message of type 'Create' -", err)
			}

		case "Connect":
			// Connect the new client
			store.Connect(request.Message.Code, request.Connection)

		default:
			// Broadcast message to all clients within the same room
			for _, conn := range store.GetConnectionsByCode(request.Message.Code) {
				conn.WriteMessage(request.Message)
			}
		}
	}
}
