package server

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type ClientMap map[string]map[*websocket.Conn]bool
type BroadcastChannel chan Request

// Server stores all connection dependencies for the websocket server.
type Server struct {
	clients ClientMap
	broadcast BroadcastChannel
	socketUpgrader websocket.Upgrader
}

// NewServer constructs a new Server instance.
func NewServer(checkOriginFunc func(r *http.Request) bool) *Server {
	return &Server{
		clients: ClientMap{},
		broadcast: make(BroadcastChannel),
		socketUpgrader: websocket.Upgrader{CheckOrigin: checkOriginFunc},
	}
}

// Start starts up the websocket server.
func (server Server) Start(port string) {
	// Concurrently deliver client messages
	go broadcastHandler(server.broadcast, server.clients)
	// Handle incoming requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		connectionHandler(w, r, server.socketUpgrader, server.broadcast)
	})

	log.Println("Started server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// connectionHandler upgrades new HTTP requests from clients to websockets, reading in further messages from
// those clients.
func connectionHandler(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader, broadcast BroadcastChannel) {
	// Upgrade HTTP GET request to a socket connection
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ERROR upgrading connection -", err)
	}
	defer socket.Close()

	// Forever handle messages from this new client
	for {
		message := ReadMessageFromJson(socket)
		if message != nil {
			broadcast <- Request{Socket: socket, Message: message}
		}
	}
}

// generateRoomCode generates a new client room code.
func generateRoomCode() string {
	return "CODE" // TODO
}

// connectClient adds the new clients connection to the ClientMap.
func connectClient(clients ClientMap, socket *websocket.Conn, roomCode string) {
	if _, v := clients[roomCode]; v {
		clients[roomCode][socket] = true
	} else {
		clients[roomCode] = map[*websocket.Conn]bool{
			socket: true,
		}
	}
}

// broadcastHandler reads in messages from a MessageChannel and forwards them on or replies to clients.
func broadcastHandler(broadcast BroadcastChannel, clients ClientMap) {
	for {
		// Pop the next message off the broadcast channel and send it
		request := <-broadcast
		switch request.Message.Type {

		case "Create":
			// Generate new room code and connect the new client
			roomCode := generateRoomCode()
			message := Message{Type: "Create", Code: roomCode}
			connectClient(clients, request.Socket, roomCode)
			SendMessage(&message, request.Socket)

		case "Connect":
			// Connect the new client
			connectClient(clients, request.Socket, request.Message.Code)

		default:
			// Read in client messages and broadcast them
			for socket := range clients[request.Message.Code] {
				SendMessage(request.Message, socket)
			}
		}
	}
}
