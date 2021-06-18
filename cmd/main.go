package main

import (
	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/server"
	"log"
	"net/http"
	"os"
)

func main() {
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
	server := NewServer(checkOrigin)
	server.Start()
}

// checkOrigin checks a requests origin, returning true if the origin is valid.
func checkOrigin(r *http.Request) bool {
	//origin := r.Header.Get("Origin") // TODO Add an origin check to the frontend url
	return true
}
