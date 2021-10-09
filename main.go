package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
)

func main() {
	useOS := len(os.Args) > 1 && os.Args[1] == "live"
	log.SetOutput(os.Stdout)
	hub := newHub()
	go hub.run()
	http.Handle("/", http.FileServer(getFileSystem(useOS)))
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	http.ListenAndServe("127.0.0.1:8080", nil)
}

// content holds our static web server content
//go:embed static
var content embed.FS

func getFileSystem(useOS bool) http.FileSystem {
	if useOS {
		log.Print("using live mode")
		return http.FS(os.DirFS("static"))
	}
	log.Print("using embed mode")
	fsys, err := fs.Sub(content, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
