package main

import (
	"fmt"
)

type Hub struct {
	//name of the hub
	name string
	// Registered clients.
	clients map[*Client]bool

	broadcast chan *Message

	register chan *Client

	unregister chan *Client
}

func newHub(name string) *Hub {
	return &Hub{
		name:       name,
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
			fmt.Printf("%s registered\n", client.id)
		case client := <-h.unregister:
			fmt.Printf("%s unregistered\n", client.id)
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			fmt.Printf("\"%s: %s\" broadcasted\n", message.Type, message.Data)
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
