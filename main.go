package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
)

type ClientMap map[string]map[*websocket.Conn]bool
type MessageChannel chan Request

// main starts up the websocket server.
func main() {
	clients := ClientMap{} // Map of room codes to client connections
	broadcast := make(MessageChannel)
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}

	// Concurrently deliver client messages
	go broadcastHandler(broadcast, clients)
	// Handle incoming requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		connectionHandler(w, r, upgrader, broadcast)
	})

	// Get the PORT
	var port string
	if len(os.Args) > 1 {
		port = os.Args[1]
	} else {
		port = os.Getenv("PORT")
	}

	if port == "" {
		log.Fatal("You must define a 'PORT' environment variable for running the web server")
	}

	log.Println("Started server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// checkOrigin checks a requests origin, returning true if the origin is valid.
func checkOrigin(r *http.Request) bool {
	//origin := r.Header.Get("Origin") // TODO Add an origin check to the frontend url
	return true
}

// connectionHandler upgrades new HTTP requests from clients to websockets, reading in further messages from
// those clients.
func connectionHandler(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader, broadcast MessageChannel) {
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
func broadcastHandler(broadcast MessageChannel, clients ClientMap) {
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

/*
 A1. Someone connects and creates a lobby; Backend returns a hash code based on the time
 A2. Person joins a socket connection; store lobby code -> socket room in memory
 A3. Wait for player B to join

 B1. Connects to a pre-existing lobby; Backend received a hash code
 B2. Backend searches store for socket room, using hash code and connects player B
 B3. Player B has joined the game

 * Game starts
	* In separate store, backend stores hash code -> lobby board
    * Sequentially backend sends a message to A,B to take their turn
    * Each turn ends with the player sending their move to the backend
    * Backend checks move is legal
*/
