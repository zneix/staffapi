package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Fetcher struct {
	httpClient    *http.Client
	helixClientID string
	helixToken    string
}

type HelixUser struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
}

type GetUsersResponse struct {
	Users []HelixUser `json:"data"`
}

func NewFetcher(clientID string, token string) *Fetcher {
	return &Fetcher{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		helixClientID: clientID,
		helixToken:    token,
	}
}

func (f *Fetcher) fetchTmiRoom(ctx context.Context, channel string, room *TmiRoom) error {
	// Create HTTP request to TMI
	tmiURL := fmt.Sprintf("https://tmi.twitch.tv/group/user/%s/chatters", channel)
	req, err := http.NewRequestWithContext(ctx, "GET", tmiURL, http.NoBody)
	if err != nil {
		return NewErrorf("Failed to create a request to TMI: %s", err)
	}

	// Execute above HTTP request
	log.Printf("[Fetch] TMI chatters in #%s\n", channel)
	res, err := f.httpClient.Do(req)
	if err != nil {
		return NewErrorf("Failed reading TMI's response: %s", err)
	}
	defer res.Body.Close()

	// Abort in case of non-200 response (which btw shouldn't happen, but this is Twitch after all...)
	if res.StatusCode != http.StatusOK {
		return NewErrorf("TMI responded with status %d", res.StatusCode)
	}

	// Serialize response body into an instance of TmiResponse
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return NewErrorf("Failed to read response body: %s, response body: %v", err, body)
	}

	if err = json.Unmarshal(body, room); err != nil {
		return NewErrorf("Failed to unmarshal tmi request: %s, response body: %s", err, string(body))
	}

	return nil
}

// fetchTwitchUsers calls Helix's Get Users to get details about users: https://dev.twitch.tv/docs/api/reference#get-users
func (f *Fetcher) fetchTwitchUsers(ctx context.Context, usernames []string) ([]*TwitchUser, error) {
	users := []*TwitchUser{}

	// TODO: Use some wg magic to make these calls concurrent
	chunks := ChunkStringSlice(usernames, 100)
	wg := new(sync.WaitGroup)
	ws := make(chan struct{}, 10)

	var err error
	for i, chunk := range chunks {
		ws <- struct{}{}
		wg.Add(1)
		go func(i int, chunk []string, w chan struct{}, wg *sync.WaitGroup) {
			defer wg.Done()
			helixURL, err := url.Parse("https://api.twitch.tv/helix/users")
			if err != nil {
				return
			}

			// Set all usernames through query parameters
			queryParams := url.Values{}
			for _, name := range chunk {
				queryParams.Add("login", name)
			}
			helixURL.RawQuery = queryParams.Encode()

			// Create Helix request
			req, err := http.NewRequestWithContext(ctx, "GET", helixURL.String(), http.NoBody)
			if err != nil {
				return
			}
			req.Header.Add("Client-ID", f.helixClientID)
			req.Header.Add("Authorization", "Bearer "+f.helixToken)

			// Execute the HTTP request(s)
			log.Printf("[Fetch] Helix %d users %d/%d\n", len(chunk), i+1, len(chunks))
			res, err := f.httpClient.Do(req)
			if err != nil {
				return
			}
			defer res.Body.Close()

			// Abort in case of non-200 response
			if res.StatusCode != http.StatusOK {
				err = NewErrorf("Helix responded with status %d", res.StatusCode)
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return
			}

			helixUsers := &GetUsersResponse{}
			err = json.Unmarshal(body, helixUsers)
			if err != nil {
				return
			}

			for _, user := range helixUsers.Users {
				users = append(users, &TwitchUser{
					Login: user.Login,
					ID:    user.ID,
					Type:  user.Type,
				})
			}
			<-w
		}(i, chunk, ws, wg)
	}
	wg.Wait()

	return users, err
}
