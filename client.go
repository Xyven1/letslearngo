package main

import (
	"log"
	"net/http"
	"regexp"
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
	Type   string            `json:"type"`
	Data   map[string]string `json:"data"`
	Status string            `json:"status"`
}

type Auth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
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
		var data Auth
		err = mapstructure.Decode(message.Data, &data)
		if err != nil {
			log.Println("Failed to parse authentication request:", err)
			break
		}
		c.send <- c.login(data)
	case "loginWithID":
		var data map[string]string
		err = mapstructure.Decode(message.Data, &data)
		if err != nil {
			log.Println("Failed to parse authentication request:", err)
			break
		}
		c.send <- c.loginWithID(data["sessionID"])
	case "register":
		var data Auth
		err = mapstructure.Decode(message.Data, &data)
		if err != nil {
			log.Println("Failed to parse authentication request:", err)
			break
		}
		c.send <- c.register(data)
	case "set":
		if err != nil {
			log.Println("Failed to parse set request:", err)
			return
		}
		c.handleSet(message.Data)
	}
}

//handles logging in a user with password
func (c *Client) login(login Auth) *Message {
	//fetch necessary data
	data := rdb.HGetAll(ctx, "user:"+login.Username).Val()
	//does user exist
	if len(data) == 0 {
		return &Message{Type: "login", Data: str2msg("Username does not exist"), Status: "error"}
	}
	//check password
	if data["password"] != login.Password {
		return &Message{Type: "login", Data: str2msg("Password is incorrect"), Status: "error"}
	}
	//login successful
	err := mapstructure.Decode(data, &c.user)
	if err != nil {
		panic(err)
	}
	//generate sessionID
	sessionID := uuid.NewString()
	rdb.HSet(ctx, "sessionIDs:"+sessionID, "username", c.user.Username)
	rdb.Expire(ctx, "sessionIDs:"+sessionID, time.Minute)
	data["sessionID"] = sessionID
	delete(data, "password")
	return &Message{Type: "login", Data: data, Status: "success"}
}

//handles logging in a user with a sessionID
func (c *Client) loginWithID(sessionID string) *Message {
	username := checkSessionID(sessionID)
	if username == "" {
		return &Message{Type: "loginWithID", Data: str2msg("SessionID is invalid"), Status: "error"}
	}
	data := rdb.HGetAll(ctx, "user:"+username).Val()
	err := mapstructure.Decode(data, &c.user)
	if err != nil {
		panic(err)
	}
	delete(data, "password")
	return &Message{Type: "loginWithID", Data: data, Status: "success"}
}

//handles registering a new user
func (c *Client) register(register Auth) *Message {
	//validate username
	err := validateUsername(register.Username)
	if err != "" {
		return &Message{Type: "register", Data: str2msg(err), Status: "error"}
	}
	//validate password
	err = validatePassword(register.Password)
	if err != "" {
		return &Message{Type: "register", Data: str2msg(err), Status: "error"}
	}
	//check if username is taken
	if val := rdb.Exists(ctx, "user:"+register.Username); val.Val() == 1 {
		return &Message{Type: "register", Data: str2msg("Username already exists!"), Status: "error"}
	}
	var redisUser map[string]string
	mapstructure.Decode(&User{Username: register.Username, Password: register.Password}, &redisUser)
	log.Println(redisUser)
	rdb.HSet(ctx, "user:"+register.Username, redisUser)
	return &Message{Type: "register", Data: str2msg("Successfully registered!"), Status: "success"}
}

//validate username
func validateUsername(username string) string {
	if len(username) < 3 {
		return "Username must at least 3 characters"
	}
	if len(username) > 20 {
		return "Maximum 20 characters"
	}
	if !regexp.MustCompile(`^[a-zA-Z]\S+$`).MatchString(username) {
		return "Username must begin with a letter"
	}
	if !regexp.MustCompile("^[a-zA-Z0-9._-]*$").MatchString(username) {
		return "Username can only contain letters, numbers, -, _, and ."
	}
	return ""
}

func validatePassword(password string) string {
	if len(password) < 8 {
		return "Password must be at least 8 characters"
	}
	return ""
}

//handle set command
func (c *Client) handleSet(setMap map[string]string) {
	for key, value := range setMap {
		rdb.Set(ctx, "public:"+key, value, 0)
	}
}

//create message map
func str2msg(message string) map[string]string {
	return map[string]string{"message": message}
}

//checks if a sessionID is valid
func checkSessionID(sessionID string) string {
	val := rdb.HGet(ctx, "sessionIDs:"+sessionID, "username").Val()
	log.Println(val)
	return val
}

//checks if user is admin based on username
func isAdmin(username string) bool {
	val := rdb.HGet(ctx, "user:"+username, "admin").Val()
	return val == "true"
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
		var message *Message = &Message{}
		err := c.conn.ReadJSON(message)
		if err != nil {
			log.Println(err)
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
