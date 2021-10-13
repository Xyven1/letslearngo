package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/websocket/v2"
)

// content holds our static web server content
//go:embed static
var content embed.FS

func main() {
	useOS := len(os.Args) > 1 && os.Args[1] == "live"
	log.SetOutput(os.Stdout)
	hub := newHub()
	go hub.run()
	go redisGlobalSub(hub, "test")

	app := fiber.New()
	if useOS {
		app.Static("/", "./static")
	} else {
		app.Use("/", filesystem.New(filesystem.Config{
			Root:       http.FS(content),
			PathPrefix: "static",
			Browse:     true,
		}))
	}
	app.Use("/api", websocket.New(func(c *websocket.Conn) {
		serveWs(hub, c)
	}))
	// http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
	// 	serveWs(hub, w, r)
	// })

	app.Listen("127.0.0.1:8081")
}
