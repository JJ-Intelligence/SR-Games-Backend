package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/JJ-Intelligence/SR-Games-Backend/pkg/server"
	"go.uber.org/zap"
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
			panic(fmt.Sprintf("Missing environment: %s", f.Name))
		}
	})
}

// checkOrigin checks a requests origin, returning true if the origin is valid.
func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return strings.Contains(origin, *frontendHost)
}

func main() {
	flag.Parse()
	checkFlagsSet()
	log, _ := zap.NewProduction()
	defer log.Sync()

	// Start-up the server
	log.Info(fmt.Sprintf("Starting server on port %s", port))
	s := server.NewServer(log, checkOrigin)
	s.Start(*port, *maxWorkers, *frontendHost)
}
