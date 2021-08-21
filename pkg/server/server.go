package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Server stores all connection dependencies for the websocket server.
type Server struct {
	Log *zap.Logger

	// LobbyStore maps Lobby IDs to Lobby structs
	Lobbys LobbyStore

	ConnToPlayerStore map[ConnectionWrapper]Player

	Upgrader websocket.Upgrader
}

// NewServer constructs a new Server instance.
func NewServer(log *zap.Logger, checkOriginFunc func(r *http.Request) bool) *Server {
	return &Server{
		Log:      log,
		Lobbys:   LobbyStore{},
		Upgrader: websocket.Upgrader{CheckOrigin: checkOriginFunc},
	}
}

// Start starts up the websocket server.
func (s *Server) Start(port string, maxWorkers int, frontendHost string) {
	// Handle incoming requests
	http.HandleFunc("/createPlayer", handlerWrapper(frontendHost, s.createPlayer()))
	http.HandleFunc("/createLobby", handlerWrapper(frontendHost, s.createLobby()))
	http.HandleFunc("/", s.connectionHandler())

	s.Log.Info(
		fmt.Sprintf(
			"Started server on port %s, with max workers %d\n",
			port, maxWorkers,
		),
	)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		s.Log.Fatal("Server errored during ListenAndServer:", zap.Error(err))
	}
}

// handlerWrapper wraps a handler to add the Access-Control-Allow-Origin header
func handlerWrapper(
	frontendHost string,
	handler func(http.ResponseWriter, *http.Request),
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//Allow CORS from frontendHost
		w.Header().Set("Access-Control-Allow-Origin", frontendHost)
		handler(w, r)
	}
}

// createPlayer returns a new player ID
func (s *Server) createPlayer() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := uuid.NewString()
		s.Log.Info("Created new Player", zap.String("playerID", id))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(id))
	}
}

// createLobby creates a new lobby, returning the lobby ID
func (s *Server) createLobby() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lobbyID := uuid.NewString()
		playerIDParam := r.URL.Query()["playerID"]

		if len(playerIDParam) == 1 {
			playerID := playerIDParam[0]
			s.Lobbys.Put(
				lobbyID,
				&Lobby{
					Host:                playerID,
					PlayerIDToConnStore: make(map[string]ConnectionWrapper),
					RequestChannel:      make(chan Request),
				},
			)
			s.Log.Info(
				"Created new Lobby",
				zap.String("lobbyID", lobbyID),
				zap.String("hostID", playerID),
			)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(lobbyID))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}
}

// connectionHandler upgrades new HTTP requests from clients to websockets,
// reading in further messages from those clients.
func (s *Server) connectionHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP GET request to a socket connection
		ws, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.Log.Info("Unable to upgrade connection", zap.Error(err))
			return
		}
		conn := &SocketWrapper{socket: ws}

		// Remove the player when their socket disconnects
		defer func() {
			conn.socket.Close()

			// Remove player
			if player, ok := s.ConnToPlayerStore[conn]; ok {
				delete(s.ConnToPlayerStore, conn)

				if lobby, ok := s.Lobbys.Get(player.LobbyID); ok {
					delete(lobby.PlayerIDToConnStore, player.PlayerID)
				}
			}
		}()

		// Handle the LobbyJoinRequest
		// TODO: Pass in LobbyJoinRequest?
		message, err := conn.ReadMessage()
		if err != nil {
			s.Log.Info(
				"Client errored before LobbyJoinRequest was sent", zap.Error(err))
			return
		}

		req := message.Contents.(LobbyJoinRequest)
		lobby, ok := s.Lobbys.Get(req.LobbyID)
		if ok {
			// Add player to lobby
			s.ConnToPlayerStore[conn] = Player{
				PlayerID: req.PlayerID,
				LobbyID:  req.LobbyID,
			}
			lobby.PlayerIDToConnStore[req.PlayerID] = conn
			s.Log.Info(
				fmt.Sprintf("Player %s joined Lobby %s", req.PlayerID, req.LobbyID))
		} else {
			s.Log.Info(
				fmt.Sprintf("Lobby %s does not exist, closing connection", req.LobbyID))
		}

		// Forever handle messages from this new client
		for {
			_, bytes, err := s.socket.ReadMessage()
			err = s.handleIncomingMessage(conn, lobby)
			if err != nil {
				s.Log.Info("Client errored or disconnected", zap.Error(err))
				return
			}
		}
	}
}

// handleIncomingMessage reads messages from a socket and sends them to the queue channel, returning an error
// if the client has disconnected.
func (s *Server) handleIncomingMessage(conn ConnectionWrapper, lobby *Lobby) error {
	message, err := conn.ReadMessage()

	if err == nil {
		lobby.RequestChannel <- Request{Connection: conn, Message: message}
	} else if _, ok := err.(*json.UnmarshalTypeError); ok {
		conn.WriteMessage(Message{
			Type:     "Error",
			Contents: "Unable to deserialise Contents to a Content Type",
		})
	}
	return err
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
