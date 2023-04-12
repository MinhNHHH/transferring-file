package main

import (
	"flag"
	"log"
)

type message struct {
	Type string
	Data interface{}
}

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
