package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisInstance struct {
	client  *redis.Client
	fetcher *Fetcher
}

// TwitchUser represents a user hash stored in redis, mapped by "user:{username}"
type TwitchUser struct {
	Login string `redis:"login" json:"login"`
	ID    string `redis:"id" json:"id"`
	Type  string `redis:"type" json:"type"`
}

// TmiRoom represents a tmi.twitch.tv response, mapped by "tmi:{channel}"
type TmiRoom struct {
	ChatterCount int `redis:"chatter_count" json:"chatter_count"`
	Chatters     struct {
		Broadcaster []string `redis:"broadcaster" json:"broadcaster"`
		VIPs        []string `redis:"vips" json:"vips"`
		Moderators  []string `redis:"moderators" json:"moderators"`
		Staff       []string `redis:"staff" json:"staff"` // could be useful(?)
		Viewers     []string `redis:"viewers" json:"viewers"`
	} `redis:"chatters" json:"chatters"`
}

func (t *TwitchUser) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TwitchUser) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func (t *TmiRoom) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TmiRoom) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, t)
}

func NewRedisInstance(url string, fetcher *Fetcher) *RedisInstance {
	opts, err := redis.ParseURL(url)
	if err != nil {
		log.Fatalln(err)
	}

	client := redis.NewClient(opts)
	return &RedisInstance{
		client,
		fetcher,
	}
}

func (r *RedisInstance) GetTwitchUsers(ctx context.Context, logins []string) ([]*TwitchUser, error) {
	users := make([]*TwitchUser, 0, len(logins))
	keys := make([]string, 0, len(logins))

	// Convert names to redis key format
	for _, login := range logins {
		keys = append(keys, "user:"+login)
	}

	res, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	// Determine which users aren't cached
	usersToFetch := make([]string, 0)

	for i, v := range res {
		if v == nil {
			// Both logins and res slices shall share the same indices
			usersToFetch = append(usersToFetch, logins[i])
			continue
		}

		user := new(TwitchUser)
		json.Unmarshal([]byte(v.(string)), user)
		//log.Printf("redis user #%d: %#v\ntwitch user #%d %#v", i+1, v, i+1, user)
		users = append(users, user)
	}

	if len(usersToFetch) != 0 {
		fetchedUsers, err := r.fetcher.fetchTwitchUsers(ctx, usersToFetch)
		if err != nil {
			return nil, err
		}

		if len(fetchedUsers) == 0 {
			log.Printf("redis: Helix didn't return any data for %d users: %s\n", len(usersToFetch), usersToFetch)
		} else {
			err = r.SetTwitchUsers(ctx, fetchedUsers)
			if err != nil {
				return nil, err
			}
			users = append(users, fetchedUsers...)
		}
	}

	return users, nil
}

func (r *RedisInstance) SetTwitchUsers(ctx context.Context, users []*TwitchUser) (err error) {
	for _, user := range users {
		xd := r.client.Set(ctx, "user:"+user.Login, user, 3*24*time.Hour)
		if err = xd.Err(); err != nil {
			log.Printf(`Error setting "user:%s": %s`, user.Login, err)
		}
	}
	return err
}

func (r *RedisInstance) GetTmiRoom(ctx context.Context, channel string) (*TmiRoom, error) {
	room := new(TmiRoom)
	err := r.client.Get(ctx, "tmi:"+channel).Scan(room)

	if err != nil {
		if err != redis.Nil {
			return nil, err
		}

		// Requested channel isn't in cache, so it has be fetched
		err := r.fetcher.fetchTmiRoom(ctx, channel, room)
		if err != nil {
			return nil, err
		}

		r.SetTmiRoom(ctx, channel, room)
		return room, nil
	}

	return room, err
}

func (r *RedisInstance) SetTmiRoom(ctx context.Context, channel string, tmiRoom *TmiRoom) error {
	return r.client.Set(ctx, "tmi:"+channel, tmiRoom, 120*time.Second).Err()
}
