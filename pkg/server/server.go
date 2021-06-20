package server

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

// Server stores all connection dependencies for the websocket server.
type Server struct {
	store          ConnectionStore
	socketUpgrader websocket.Upgrader
}

// NewServer constructs a new Server instance.
func NewServer(checkOriginFunc func(r *http.Request) bool) *Server {
	return &Server{
		store:          NewMapConnectionStore(),
		socketUpgrader: websocket.Upgrader{CheckOrigin: checkOriginFunc},
	}
}

// Start starts up the websocket server.
func (s Server) Start(port string, maxWorkers int) {
	// Create a RequestHandler to concurrently deliver client messages
	requests := make(RequestChannel)
	requestHandler := NewRequestHandler(requests, maxWorkers)
	go requestHandler.Start(s.store)

	// Handle incoming requests
	http.HandleFunc("/", connectionHandler(s.store, requests, s.socketUpgrader))

	log.Printf("Started server on port %s, with max workers %d\n", port, maxWorkers)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ERROR server failed during ListenAndServer -", err)
	}
}

// connectionHandler upgrades new HTTP requests from clients to websockets, reading in further messages from
// those clients.
func connectionHandler(
	store ConnectionStore,
	requests RequestChannel,
	upgrader websocket.Upgrader,
) func(w http.ResponseWriter, r *http.Request) {

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
			err = handleIncomingMessage(socket, requests)
			if err != nil {
				log.Println("Client errored or disconnected", err)
				return
			}
		}
	}
}

// handleIncomingMessage reads messages from a socket and sends them to the queue channel, returning an error
// if the client has disconnected.
func handleIncomingMessage(conn ConnectionWrapper, requests RequestChannel) error {
	message, err := conn.ReadMessage()

	if websocket.IsUnexpectedCloseError(err) {
		log.Println("Client errored or disconnected", err)
		return err
	} else if err != nil {
		log.Println("ERROR reading incoming message -", err)
	} else {
		requests <- Request{Connection: conn, Message: message}
	}
	return nil
}

type RequestChannel chan Request               // Channel of incoming client requests
type WorkerRequestChannels chan RequestChannel // Channel of request channels belonging to each worker

// RequestHandler stores request and worker channels for concurrently handling incoming client requests.
type RequestHandler struct {
	requests RequestChannel
	workers  WorkerRequestChannels
}

func NewRequestHandler(requests RequestChannel, maxWorkers int) *RequestHandler {
	return &RequestHandler{
		requests: requests,
		workers:  make(WorkerRequestChannels, maxWorkers),
	}
}

// Start creates concurrent worker functions which handle incoming requests, and passes incoming requests to
// free workers.
func (h *RequestHandler) Start(store ConnectionStore) {
	// Create workers
	for i := 0; i < cap(h.workers); i++ {
		go runRequestWorker(h.workers, make(RequestChannel), store)
	}

	// Pass incoming requests to workers
	for {
		req := <-h.requests

		// Concurrently find a free worker and add this request to their RequestChannel
		go func() {
			worker := <-h.workers
			worker <- req
		}()
	}
}

// runRequestWorker forever handles requests from the given RequestChannel.
func runRequestWorker(workers WorkerRequestChannels, requests RequestChannel, store ConnectionStore) {
	for {
		// Register as a worker
		workers <- requests

		// Wait for a request to handle
		r := <-requests

		switch r.Message.Type {
		case "Create":
			// Generate new room code and connect the new client
			code := store.NewCode()
			store.Connect(code, r.Connection)
			err := r.Connection.WriteMessage(Message{Type: "Create", Code: code})
			if err != nil {
				log.Println("ERROR sending message of type 'Create' -", err)
			}

		case "Connect":
			// Connect the new client
			store.Connect(r.Message.Code, r.Connection)

		default:
			// Broadcast message to all clients within the same room
			for _, conn := range store.GetConnectionsByCode(r.Message.Code) {
				err := conn.WriteMessage(r.Message)
				if err != nil {
					log.Printf("ERROR sending message of type '%s' - %s\n", r.Message.Type, err)
				}
			}
		}
	}
}
