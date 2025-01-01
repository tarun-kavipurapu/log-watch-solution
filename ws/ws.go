package ws

import (
	"log"
	"test/watch"
	"time"

	"github.com/gorilla/websocket"
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

// Client represents a single WebSocket connection.
type Client struct {
	conn   *websocket.Conn
	sendTo chan []byte
	hub    *Hub
}

func (c *Client) readPump() {

	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	logchan := watch.GetLogChan()
	for {
		msg := <-logchan
		c.hub.broadcast <- msg
	}
}

// WritePump handles sending messages to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.sendTo:
			// log.Println(string(message))
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed, close the connection
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Close()

		case <-ticker.C:
			// Send ping periodically
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func CreateClient(conn *websocket.Conn, hub *Hub) {
	client := &Client{
		conn:   conn,
		sendTo: make(chan []byte, 256),
		hub:    hub,
	}
	hub.register <- client
	watcher, err := watch.GetWatcher()
	if err != nil {
		log.Println(err)
	}
	if err := watcher.SendbottomLines(); err != nil {
		log.Println(err)
	}

	go client.WritePump()
	go client.readPump()
}
