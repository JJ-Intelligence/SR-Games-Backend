package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/server"
)

var (
	port         = flag.String("port", os.Getenv("PORT"), "Port to host the server on")
	maxWorkers   = flag.Int("maxWorkers", getEnvOrDefault("MAX_WORKERS", 10).(int), "Maximum number of workers handling socket requests")
	frontendHost = flag.String("frontendHost", os.Getenv("FRONTEND_HOST"), "The frontend host")
)

// getEnvOrDefault tries to get an Environment variable or returns a default
// if it doesn't exist
func getEnvOrDefault(key string, def interface{}) interface{} {
	env, ok := os.LookupEnv(key)
	if ok {
		return env
	}
	return def
}

// checkFlagsSet will panic if a flag has not been set
func checkFlagsSet() {
	flag.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "" {
			log.Fatal(fmt.Sprintf("Missing environment: %s", f.Name))
		}
	})
}

// checkOrigin checks a requests origin, returning true if the origin is valid.
func checkOrigin(r *http.Request) bool {
	// log.Println(r.RemoteAddr)
	// return strings.Contains(r.RemoteAddr, *frontendHost)
	return true
}

func main() {
	flag.Parse()
	checkFlagsSet()

	// Start-up the server
	s := server.NewServer(checkOrigin)
	s.Start(*port, *maxWorkers, *frontendHost)
}
