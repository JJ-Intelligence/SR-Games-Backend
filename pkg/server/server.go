package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/comms"
	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/lobby"
	"github.com/mitchellh/mapstructure"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	CHANNEL_BUFFER_LEN = 10
	PING_TIMEOUT       = 50 * time.Second
)

// Server stores all connection dependencies for the websocket server.
type Server struct {
	Log *zap.Logger

	// LobbyStore maps Lobby IDs to Lobby structs
	Lobbys lobby.LobbyStore

	ConnToPlayerStore map[*comms.ConnectionWrapper]lobby.Player

	Upgrader websocket.Upgrader
}

// NewServer constructs a new Server instance.
func NewServer(log *zap.Logger, checkOriginFunc func(r *http.Request) bool) *Server {
	return &Server{
		Log:               log,
		Lobbys:            lobby.LobbyStore{},
		ConnToPlayerStore: make(map[*comms.ConnectionWrapper]lobby.Player),
		Upgrader:          websocket.Upgrader{CheckOrigin: checkOriginFunc},
	}
}

// Start starts up the websocket server.
func (s *Server) Start(port string, maxWorkers int, frontendHost string) {
	// Handle incoming requests
	http.HandleFunc("/createPlayer", handlerWrapper(frontendHost, s.createPlayer()))
	http.HandleFunc("/createLobby", handlerWrapper(frontendHost, s.createLobby()))
	http.HandleFunc("/", s.connectionReadHandler())

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

		if len(playerIDParam) == 1 && lobby.IsValidPlayerID(playerIDParam[0]) {
			playerID := playerIDParam[0]
			l := &lobby.Lobby{
				Log:                 s.Log,
				Host:                playerID,
				PlayerIDToConnStore: make(map[string]*comms.ConnectionWrapper),
				RequestChannel:      make(chan comms.Request, CHANNEL_BUFFER_LEN),
			}
			s.Lobbys.Put(lobbyID, l)
			go l.LobbyRequestHandler()

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

// connectionReadHandler upgrades new HTTP requests from clients to websockets,
// reading in further messages from those clients.
func (s *Server) connectionReadHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade HTTP GET request to a socket connection
		ws, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.Log.Info("Unable to upgrade connection", zap.Error(err))
			return
		}
		conn := &comms.ConnectionWrapper{
			Socket:       ws,
			WriteChannel: make(chan comms.Message, CHANNEL_BUFFER_LEN),
		}

		// Remove the player when their socket disconnects
		defer func() {
			conn.Close()

			if player, ok := s.ConnToPlayerStore[conn]; ok {
				delete(s.ConnToPlayerStore, conn)
				if lobby, ok := s.Lobbys.Get(player.LobbyID); ok {
					delete(lobby.PlayerIDToConnStore, player.PlayerID)

					// Close the lobby if this is the host
					if lobby.Host == player.PlayerID {
						s.Log.Info(fmt.Sprintf("Closing lobby %s", player.LobbyID))
						lobby.Close()
						s.Lobbys.Delete(player.LobbyID)
					}
				}
			}
		}()

		// Start up writer process
		go s.connectionWriteHandler(conn)

		// Wait for a successful LobbyJoinRequest
		var (
			l        *lobby.Lobby
			playerID string
		)
		err = s.parseMessageLoop(conn, func(message comms.Message) (bool, error) {
			// Wait for a LobbyJoinRequest
			if message.Type != "LobbyJoinRequest" {
				conn.WriteChannel <- comms.ToMessage(comms.ErrorResponse{
					Reason: fmt.Sprintf(
						"First message should be a LobbyJoinRequest but was %s", message.Type),
				})
			} else {
				// Parse the Message contents to a LobbyJoinRequest
				var req lobby.LobbyJoinRequest
				err = mapstructure.Decode(message.Contents, &req)

				if err == nil {
					if lobby.IsValidPlayerID(req.PlayerID) {
						// Check if the lobby exists
						l, ok := s.Lobbys.Get(req.LobbyID)
						if ok {
							// Add the player to the lobby if it exists
							conn.PlayerID = req.PlayerID
							s.ConnToPlayerStore[conn] = lobby.Player(req)
							l.PlayerIDToConnStore[req.PlayerID] = conn
							l.RequestChannel <- comms.Request{
								ConnChannel: conn.WriteChannel,
								PlayerID:    req.PlayerID,
								Message:     comms.ToMessage(lobby.PlayerJoinedEvent{}),
							}
							s.Log.Info(
								fmt.Sprintf(
									"Player %s joined Lobby %s",
									req.PlayerID, req.LobbyID,
								),
							)
							return false, nil
						} else {
							conn.WriteChannel <- comms.ToMessage(comms.ErrorResponse{
								Reason: fmt.Sprintf("Lobby %s does not exist", req.LobbyID),
							})
						}
					} else {
						conn.WriteChannel <- comms.ToMessage(comms.ErrorResponse{
							Reason: fmt.Sprintf("Invalid player ID %s", req.PlayerID),
						})
					}
				} else {
					conn.WriteChannel <- comms.ToMessage(comms.ErrorResponse{
						Reason: "Unable to parse message contents to LobbyJoinRequest",
					})
				}
			}
			return true, nil
		})
		if err != nil {
			return
		}

		// Keep websocket conection alive by sending a ping every 50 seconds
		// (Heroku closes connections after 55s)
		pingAfterTimeout(conn)

		// Read in messages and push them onto the Lobby RequestChannel
		s.parseMessageLoop(conn, func(message comms.Message) (bool, error) {
			switch message.Type {
			case "LobbyLeaveRequest":
				delete(l.PlayerIDToConnStore, playerID)
				l.RequestChannel <- comms.Request{
					ConnChannel: conn.WriteChannel,
					PlayerID:    playerID,
					Message:     comms.ToMessage(lobby.PlayerLeftEvent{}),
				}
				return false, nil
			default:
				l.RequestChannel <- comms.Request{
					ConnChannel: conn.WriteChannel,
					PlayerID:    playerID,
					Message:     message,
				}
				return true, nil
			}
		})
	}
}

func pingAfterTimeout(conn *comms.ConnectionWrapper) {
	time.AfterFunc(PING_TIMEOUT, func() {
		err := conn.WriteMessage(comms.ToMessage(comms.Ping{}))
		if err == nil {
			pingAfterTimeout(conn)
		}
	})
}

func (s *Server) parseMessageLoop(
	conn *comms.ConnectionWrapper,
	parseMessageCB func(message comms.Message) (bool, error),
) error {
	for {
		message, err := conn.ReadMessage()

		if err != nil {
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				conn.WriteChannel <- comms.ToMessage(comms.ErrorResponse{
					Reason: "Unable to deserialise message",
					Error:  err,
				})
			} else {
				s.Log.Info("Client errored or disconnected", zap.Error(err))
				return err
			}
		} else {
			if ok, err := parseMessageCB(message); !ok {
				return err
			}
		}
	}
}

func (s *Server) connectionWriteHandler(conn *comms.ConnectionWrapper) {
	for {
		message := <-conn.WriteChannel
		conn.WriteMessage(message)

		if _, ok := message.Contents.(lobby.LobbyClosedBroadcast); ok {
			conn.Socket.Close()
			return
		}
	}
}
