package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var start time.Time

// Client is a struct which contains data bout a client including the websocket connection and hub
type Client struct {
	id   string
	ip   string
	user *User
	hub  *Hub
	conn *websocket.Conn
	send chan *Message
}

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, ip: r.RemoteAddr, id: uuid.NewString(), send: make(chan *Message)}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// parses a message and sends it to the correct handler
func (c *Client) handleMessage(message *Message) {
	//parse messages from client and deal with them
	var err error
	switch message.Type {
	case "message":
		c.hub.broadcast <- message
	case "login":
		var login Auth
		err = json.Unmarshal([]byte(message.Data), &login)
		if err != nil {
			log.Println("Failed to parse authentication request:", err)
			break
		}
		c.send <- c.login(login)
	case "register":
		var register Auth
		err = json.Unmarshal([]byte(message.Data), &register)
		if err != nil {
			log.Println("Failed to parse authentication request:", err)
			break
		}
		c.send <- c.register(register)
	}
}

//hanldes logining in a user
func (c *Client) login(login Auth) *Message {
	//fetch necessary data
	data := rdb.HGetAll(ctx, "user:"+login.Username).Val()
	//does user exist
	if len(data) == 0 {
		return &Message{Type: "login", Data: "Username does not exist"}
	}
	//check password
	if data["password"] != login.Password {
		return &Message{Type: "login", Data: "Password is incorrect"}
	}
	//login successful
	err := mapstructure.Decode(data, &c.user)
	if err != nil {
		panic(err)
	}
	return &Message{Type: "login", Data: "Success"}
}

//handles registering a new user
func (c *Client) register(register Auth) *Message {
	if val := rdb.Exists(ctx, "user:"+register.Username); val.Val() == 1 {
		return &Message{Type: "register", Data: "Username already exists!"}
	}
	var redisUser map[string]interface{}
	mapstructure.Decode(&User{Username: register.Username, Password: register.Password}, &redisUser)
	log.Println(redisUser)
	rdb.HSet(ctx, "user:"+register.Username, redisUser)
	return &Message{Type: "register", Data: "Successfully registered!"}
}

//readPump recieves messages from the websocket connection and handles them
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var message *Message
		err := c.conn.ReadJSON(message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		start = time.Now()
		c.handleMessage(message)
	}
}

//writePump sends messages through the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.WriteJSON(message)
			for i := 0; i < len(c.send); i++ {
				c.conn.WriteJSON(<-c.send)
			}
			log.Println("Request took:", time.Since(start))
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
