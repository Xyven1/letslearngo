package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

func redisGlobalSub(hub *Hub, key string) {
	channel := rdb.Subscribe(ctx, fmt.Sprintf("__keyspace@0__:%s", key)).Channel()
	for {
		msg, ok := <-channel
		if !ok {
			return
		}
		switch msg.Payload {
		case "set":
			hub.broadcast <- &Message{Type: "set", Data: str2msg(rdb.Get(ctx, "name").Val())}
		}
	}
}
func redisSub(c *Client, key string) {
	channel := rdb.Subscribe(ctx, fmt.Sprintf("__keyspace@0__:%s", key)).Channel()
	for {
		msg, ok := <-channel
		if !ok {
			return
		}
		switch msg.Payload {
		case "set":
			c.send <- &Message{Type: "set", Data: str2msg(rdb.Get(ctx, "name").Val())}
		}
	}
}
