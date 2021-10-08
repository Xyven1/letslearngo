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
var subscriber = rdb.Subscribe(ctx, "__keyspace@0__:name").Channel()

func redisSub(hub *Hub) {
	for {
		msg := <-subscriber
		fmt.Println("Test:" + msg.Payload)
		fmt.Println("Test:" + msg.Channel)
		switch msg.Payload {
		case "set":
			hub.broadcast <- &Message{Type: "set", Data: rdb.Get(ctx, "name").Val()}
		case "del":
			fmt.Println("del")
		default:
			fmt.Println("default")
		}
	}
}
