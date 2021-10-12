package main

import (
	"log"
)

type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	broadcast chan *Message

	register chan *Client

	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		register:   make(chan *Client),
		broadcast:  make(chan *Message),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("%s registered\n", client.uuid)
		case client := <-h.unregister:
			log.Printf("%s unregistered\n", client.uuid)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			// log.Printf("\"%s: %s\" broadcasted\n", message.Type, message.Data)
			// rdb.XAdd(ctx, &redis.XAddArgs{Stream: "chatHistory", Values: message.Data})
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
