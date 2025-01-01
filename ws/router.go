package ws

import (
	"log"
	"net/http"
	"test/watch"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func SetupRoutes(server *Server) *gin.Engine {
	r := server.router
	watcher, err := watch.GetWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}

	go watcher.WatchFile()
	hub := newHub()
	go hub.run()
	r.GET("/log", func(c *gin.Context) {
		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		connection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println(err)
			return
		}

		CreateClient(connection, hub)
	})

	return r
}
