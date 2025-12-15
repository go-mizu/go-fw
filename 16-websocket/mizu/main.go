package main

import (
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	app := mizu.New()

	app.Get("/ws", func(c *mizu.Ctx) error {
		conn, err := upgrader.Upgrade(c.Writer(), c.Request(), nil)
		if err != nil {
			return err
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return nil
			}
			conn.WriteMessage(mt, msg)
		}
	})

	app.Listen(":8080")
}
