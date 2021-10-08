package main

import (
	"fmt"
)

var subscriber = rdb.Subscribe(ctx, "__keyspace@0__:name").Channel()

func runSub(hub *Hub) {
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
