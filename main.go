package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
)

type ClientMap map[string]map[*websocket.Conn]bool
type MessageChannel chan Message

func checkOrigin(r *http.Request) bool {
	//origin := r.Header.Get("Origin") // TODO Add an origin check to the frontend
	return true
}

func main() {
	clients := ClientMap{} // Map of room codes to client connections
	broadcast := make(MessageChannel)
	upgrader := websocket.Upgrader{CheckOrigin: checkOrigin}

	// Handle incoming requests
	http.HandleFunc("/", connectionHandler)
	// Concurrently deliver client messages
	go broadcastHandler(broadcast, clients)

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

// getRoomCode extracts the room code from the HTTP request query or generates a new room code.
func getRoomCode(r *http.Request) string {
	roomCode :=
	if roomCode == "" {
		// Generate a room code
		roomCode = "CODE" // TODO How do we generate the room code?
	}

	return roomCode
}

func connectionHandler(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader, broadcast MessageChannel, clients ClientMap) {
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
			broadcast <- *message
		}
	}
}

func generateRoomCode() string {
	return "CODE"
}

func broadcastHandler(broadcast MessageChannel, clients ClientMap) {
	for {
		// Pop the next message off the broadcast channel and send it
		message := <-broadcast
		for socket := range clients[message.Code] {
			switch message.Type {
			case "GenerateRoomCode":
				roomCode := generateRoomCode()
				message = Message{Type: "GenerateRoomCode", Code: roomCode}
			}



			// Get the room code
			roomCode := r.URL.Query().Get("roomCode")

			// Update client map
			if _, v := clients[roomCode]; v {
				clients[roomCode][socket] = true
			} else {
				clients[roomCode] = map[*websocket.Conn]bool{
					socket: true,
				}
			}

			SendMessage(&message, socket)
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
