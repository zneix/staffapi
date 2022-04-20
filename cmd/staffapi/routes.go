package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-z0-9](?:\w{1,24})?$`)

// handleStaff handles GET /
func (server *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleStaff handles GET /staff
func (server *Server) handleStaff(w http.ResponseWriter, r *http.Request) {
	channel := strings.ToLower(r.URL.Query().Get("channel"))

	if !usernameRegex.MatchString(channel) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make a Redis call and get cached tmi chatters (or call TMI API and set response in redis)
	tmiRoom, err := server.redis.GetTmiRoom(r.Context(), channel)
	if err != nil {
		log.Printf("failed to get tmiRoom: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Early-out if there's nothing for us to check
	if tmiRoom.ChatterCount == 0 {
		fmt.Fprintf(w, "No staff in %s TriHard\n", channel)
		return
	}

	// Handle all chatters from tmiRoom
	staff := make([]string, 0)
	names := make([]string, 0, tmiRoom.ChatterCount)

	for _, name := range tmiRoom.Chatters.Broadcaster {
		names = append(names, name)
	}
	for _, name := range tmiRoom.Chatters.Moderators {
		names = append(names, name)
	}
	for _, name := range tmiRoom.Chatters.VIPs {
		names = append(names, name)
	}
	for _, name := range tmiRoom.Chatters.Viewers {
		names = append(names, name)
	}

	// Make a Redis call and get cached values (or call Helix API and set Helix's result in redis)
	users, err := server.redis.GetTwitchUsers(r.Context(), names)
	if err != nil {
		log.Printf("err getting redis users: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check for staff members
	for _, user := range users {
		if user.Type == "staff" || user.Type == "admin" {
			staff = append(staff, user.Login)
		}
	}

	// Write HTTP response
	if len(staff) == 0 {
		fmt.Fprintf(w, "No staff in %s TriHard\n", channel)
		return
	}

	fmt.Fprintf(w, "monkaS there's %d staff members in chat: %s\n", len(staff), strings.Join(staff, " "))
}

func registerRoutes(server *Server) {
	server.mux.HandleFunc("/", server.handleIndex)
	server.mux.HandleFunc("/staff", server.handleStaff)
}
