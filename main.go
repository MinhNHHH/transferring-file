package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

var (
	upgrader = websocket.Upgrader{
		// ReadBufferSize:  1024,
		// WriteBufferSize: 1024,
		// CheckOrigin: func(r *http.Request) bool {
		// 	return true
		// },
	}

	clients = make(map[*websocket.Conn]bool)
)

func main() {
	http.HandleFunc("/", handleWebSocket)
	err := http.ListenAndServe(":8080", nil)
	log.Println("Listening Sever port: 8080")
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket connection:", err)
		return
	}
	defer conn.Close()

	// Add new client to clients map
	clients[conn] = true

	for {
		// Read incoming message from client
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Error reading JSON message from client:", err)
			delete(clients, conn)
			break
		}
		// Broadcast message to all connected clients
		for client := range clients {
			if client != conn {
				err := client.WriteJSON(msg)
				if err != nil {
					log.Println("Error writing JSON message to client:", err)
					delete(clients, client)
					break
				}
			}
		}
	}
}
