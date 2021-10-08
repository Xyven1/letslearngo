package main

import (
	"net/http"
)

type Form struct {
	Name string
}

func main() {
	hub := newHub()
	go hub.run()
	go redisSub(hub)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.ListenAndServe("127.0.0.1:8080", nil)
}
