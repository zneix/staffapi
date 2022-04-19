package main

import (
	"log"
	"os"
)

func init() {
	log.SetFlags(log.Flags() | log.Lmicroseconds)
}

func main() {
	clientID := os.Getenv("HELIX_CLIENTID")
	token := os.Getenv("HELIX_TOKEN")
	if clientID == "" || token == "" {
		log.Fatalln("Both HELIX_CLIENTID and HELIX_TOKEN env vars have to be set!")
	}

	server := NewServer(":2559")
	fetcher := NewFetcher(clientID, token)
	redis := NewRedisInstance("redis://localhost:6379/3", fetcher)

	server.redis = redis
	server.listen() // blocking operation
}
