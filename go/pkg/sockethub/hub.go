package sockethub

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"

	"nathejk.dk/pkg/job"
)

type hub struct {
	db   *redis.Client
	jobs *job.Runner
	// Registered clients.
	clients map[*Client]bool

	// messages to the clients.
	broadcast chan message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub(db *redis.Client) *hub {
	m := &hub{
		broadcast:  make(chan message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		db:         db,
		jobs:       job.NewRunner(),
	}
	go m.jobs.Run()
	go m.run()

	return m
}

func (h *hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if client.send != nil {
					close(client.send)
				}
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (h *hub) Read(view string, key string, body interface{}) error {
	jsonstr, err := h.db.HGet(view, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(jsonstr), body)
}

func (h *hub) Update(view string, key string, body interface{}) {
	h.broadcast <- message{
		View: view,
		Key:  key,
		Body: body,
	}
	h.jobs.Add(key, func() {
		if body == nil {
			h.db.HDel(view, key).Result()
			return
		}
		value, _ := json.Marshal(body)
		_, err := h.db.HSet(view, key, value).Result()
		if err != nil && err != redis.Nil {
			panic(err)
		}
	})
}

// Configure the upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//	handleConnection(h, w, r)
	//}
	//func handleConnection(h *hub, w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(ws, h)
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.Write()
	go client.Listen()
}
