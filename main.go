package main

import (
	transferringfile "webrtc/cli/pkg/transferring-file"
)

type message struct {
	Type string
	Data interface{}
}

func main() {
	rc := transferringfile.NewRemoteClient()

	rc.Connect("localhost:8080", "1")
}
