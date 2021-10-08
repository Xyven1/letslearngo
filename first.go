package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/schema"
)

type Form struct {
	Name string
}

var decoder = schema.NewDecoder()
var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

var hubs = make(map[string]*Hub)

func main() {
	hub := newHub("1")
	hubs[hub.name] = hub
	go hub.run()
	go runSub(hub)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api", api)
	http.HandleFunc("/socket", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.ListenAndServe("127.0.0.1:8080", nil)
}

//rest api
func api(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "Value: %s", rdb.Get(ctx, "name").Val())
	case "POST":
		err := r.ParseForm()
		if err != nil {
			fmt.Println(err)
		}
		var form Form
		err = decoder.Decode(&form, r.PostForm)
		if err != nil {
			fmt.Println(err)
		}
		rdb.Set(ctx, "name", form.Name, 0)
		fmt.Printf("Hello %s\n", form.Name)
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported")
	}
}

// func socketApi(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	defer conn.Close()
// 	for {
// 		_, msg, err := conn.ReadMessage()
// 		if err != nil {
// 			fmt.Println(err)
// 			return
// 		}
// 		rdb.Set(ctx, "name", string(msg), 0)
// 		fmt.Printf("Hello %s\n", string(msg))
// 	}
// }
