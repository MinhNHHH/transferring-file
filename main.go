package main

import (
	"flag"
	"log"

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
	var server = flag.String("server", "localhost:3001", "Address to signaling server")
	var noTurn = flag.Bool("no-turn", false, "Don't use a TURN server")

	flag.Parse()
	args := flag.Args()

	if len(args) == 1 {
		log.Println(args)
	} else {
		log.Printf(*server)
		log.Println(noTurn)
		log.Println(args)
	}
}
