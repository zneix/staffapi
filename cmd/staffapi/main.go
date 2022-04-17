package main

import (
	"log"
)

func init() {
	log.SetFlags(log.Flags() | log.Lmicroseconds)
}

func main() {
	server := NewServer(":2559")
	server.listen() // blocking operation
}
