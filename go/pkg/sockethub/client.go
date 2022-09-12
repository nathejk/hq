// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sockethub

import (
	"encoding/json"
	"log"
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

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *hub

	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send  chan message
	views map[string]filter
}

func NewClient(ws *websocket.Conn, hub *hub) *Client {
	return &Client{
		hub:  hub,
		ws:   ws,
		send: make(chan message),

		//send: make(chan message, 256),
	}
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) Listen() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		var msg subscribeMessage
		err := c.ws.ReadJSON(&msg)
		//typ, m, err := c.ws.ReadMessage()
		log.Printf("Got msg %#v", msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}

			log.Printf("error: %v", err)
			break
		}
		/*
			body, err := c.hub.db.HGet(msg.View, msg.Key).Result()
			if err != nil {
					log.Printf("error: %v", err)
			}*/

		list, err := c.hub.db.HGetAll(msg.View).Result()
		for key, jsonstr := range list {
			var body interface{}
			if err := json.Unmarshal([]byte(jsonstr), &body); err == nil {
				c.send <- message{
					View: msg.View,
					Key:  key,
					Body: body,
				}
			}
		}

		//c.hub.broadcast <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.ws.WriteJSON(message)
			/*
				w, err := c.ws.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				w.Write(message)

				// Add queued chat messages to the current websocket message.
				n := len(c.send)
				for i := 0; i < n; i++ {
					w.Write(newline)
					w.WriteJSON(<-c.send)
				}

				if err := w.Close(); err != nil {
					return
				}*/
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
