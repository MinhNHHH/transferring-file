package main

import (
	"flag"
	transferringfile "webrtc/cli/pkg/transferring-file"
)

type message struct {
	Type string
	Data interface{}
}

func main() {
	var server = flag.String("server", "ws://localhost:3001", "Address to signaling server")
	var noTurn = flag.Bool("no-turn", true, "Don't use a TURN server")

	flag.Parse()
	args := flag.Args()

	if len(args) == 1 {
		// rc := transferringfile.RemoteClient()

	} else {
		tf := transferringfile.New(*noTurn)
		tf.Start(*server)
		return
	}
}
