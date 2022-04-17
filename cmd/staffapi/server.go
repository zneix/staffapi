package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Server an instance wrapping everything that is responsible for handling incoming HTTP requests and querying necessary data
type Server struct {
	httpClient *http.Client
	httpServer *http.Server
	mux        *chi.Mux
	address    string
}

// NewServer creates a new Server instance, wrapping http.Server and http.Client which will be used while preparing requests
func NewServer(address string) *Server {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	mux := chi.NewMux()

	httpServer := &http.Server{
		Addr:         address,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	server := &Server{
		httpClient: client,
		httpServer: httpServer,
		mux:        mux,
		address:    address,
	}

	registerRoutes(server)
	return server
}

// listen makes s listen on s.address which is a blocking operation
func (s *Server) listen() {
	log.Printf("[API] Listening on %s\n", s.address)
	log.Fatalln(s.httpServer.ListenAndServe())
}
