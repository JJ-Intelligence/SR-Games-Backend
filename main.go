package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
)

type Message struct {
	Type     string `json:"type"`
	Code     string `json:"code"`
	Contents string `json:"message,omitempty"`
}

var clients = map[string]map[*websocket.Conn]bool{} // Map of room codes to client connections
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
	//origin := r.Header.Get("Origin") // TODO Add an origin check to the frontend
	return true
}}

func main() {
	// Handle incoming requests
	http.HandleFunc("/", connectionHandler)
	// Concurrently deliver client messages
	go broadcastHandler()

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

/* handleRoomCode extracts the room code from the HTTP request query or generates a room code and
 * sends it to the client.
 */
func handleRoomCode(r *http.Request, socket *websocket.Conn) string {
	roomCode := r.URL.Query().Get("roomCode")
	if roomCode == "" {
		// Generate a room code
		roomCode = "CODE" // TODO How do we generate the room code?
		_ = socket.WriteJSON(Message{
			Type: "RoomCode",
			Code: roomCode,
		})
	}

	return roomCode
}

func connectionHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP GET request to a socket connection
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ERROR upgrading connection -", err)
	}
	defer socket.Close()

	// Get the room code
	roomCode := handleRoomCode(r, socket)

	// Update client map
	if _, v := clients[roomCode]; v {
		clients[roomCode][socket] = true
	} else {
		clients[roomCode] = map[*websocket.Conn]bool{
			socket: true,
		}
	}

	// Forever handle messages from this new client
	for {
		var message Message
		_ = socket.ReadJSON(&message)
		broadcast <- message // Send message to broadcast channel
	}
}

func broadcastHandler() {
	for {
		// Pop the next message off the broadcast channel and send it
		message := <-broadcast
		for socket := range clients[message.Code] {
			_ = socket.WriteJSON(message)
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
