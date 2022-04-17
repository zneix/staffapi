package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

type RedisInstance struct {
	client *redis.Client
}

type TwitchUserData struct {
	Username string
	UserID   string
	Type     string // could be "", "staff", "admin", etc.
}

func NewRedisClient() *RedisInstance {
	opts, err := redis.ParseURL("redis://localhost:6379/3")
	if err != nil {
		log.Fatalln(err)
	}

	client := redis.NewClient(opts)
	return &RedisInstance{
		client: client,
	}
}

func (r *RedisInstance) GetJSON(ctx context.Context, key string) {
	res := r.client.Get(ctx, key)
	fmt.Printf("result from redis.Get call: %#v\n", res)
}

func (r *RedisInstance) SetJSON(ctx context.Context) {
	//
}
