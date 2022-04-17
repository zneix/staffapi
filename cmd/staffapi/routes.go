package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var usernameRegex = regexp.MustCompile(`^[a-z0-9](?:\w{1,24})?$`)

// TmiResponse represents a tmi.twitch.tv response
type TmiResponse struct {
	Chatters struct {
		Broadcaster []string `json:"broadcaster"`
		VIPs        []string `json:"vips"`
		Moderators  []string `json:"moderators"`
		//Staff       []string `json:"staff"` // could be useful(?)
		Viewers []string `json:"viewers"`
	} `json:"chatters"`
}

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

	// Create HTTP request to tmi API
	tmiURL := fmt.Sprintf("https://tmi.twitch.tv/group/user/%s/chatters", channel)
	req, err := http.NewRequestWithContext(r.Context(), "GET", tmiURL, http.NoBody)
	if err != nil {
		log.Printf("Error while making tmi request: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Execute HTTP request created above
	res, err := server.httpClient.Do(req)
	if err != nil {
		log.Printf("Error while reading tmi's response: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()

	// Abort in case of non-200 response (which btw shouldn't happen, but this is Twitch after all...)
	if res.StatusCode != http.StatusOK {
		log.Printf("Received status code %d from a tmi request, aborting", res.StatusCode)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Serialize response body into an instance of TmiResponse
	body, err := io.ReadAll(res.Body)
	var tmiResponse TmiResponse
	if json.Unmarshal(body, &tmiResponse) != nil {
		log.Printf("Failed to unmarshal tmi request: %s, tmi request body: %s\n", err, string(body))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Handle all chatters from tmiResponse and form a slice with their usernames
	usernames := []string{}

	for _, v := range tmiResponse.Chatters.Broadcaster {
		usernames = append(usernames, v)
	}
	for _, v := range tmiResponse.Chatters.Moderators {
		usernames = append(usernames, v)
	}
	for _, v := range tmiResponse.Chatters.VIPs {
		usernames = append(usernames, v)
	}
	for _, v := range tmiResponse.Chatters.Viewers {
		usernames = append(usernames, v)
	}
	fmt.Printf("Received a list of %d chatters: %s\n", len(usernames), usernames)

	// TODO: make a Redis call and get (or call Helix API and set Helix's result if users aren't cached)

	// just give some response and eShrug
	fmt.Fprintf(w, "No staff in %s TriHard\n", channel)
}

func registerRoutes(server *Server) {
	server.mux.HandleFunc("/", server.handleIndex)
	server.mux.HandleFunc("/staff", server.handleStaff)
}
