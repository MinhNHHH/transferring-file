package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	transferringfile "github.com/minhnh/transfer-file/cli/pkg/transferring-file"
)

type message struct {
	Type string
	Data interface{}
}

func main() {
	var server = flag.String("server", "http://localhost:3001", "Address to signaling server")
	var noTurn = flag.Bool("no-turn", true, "Don't use a TURN server")
	var filePath string = ""
	flag.Parse()
	args := flag.Args()

	if len(args) == 1 {
		rc := transferringfile.NewRemoteClient()

		serverURLRe := regexp.MustCompile(`^((http|https):\/\/[^\s/]+)\/([^\s/]+)*`)
		matches := serverURLRe.FindSubmatch([]byte(args[0]))

		if len(matches) == 4 {
			rc.Connect(string(matches[1]), string(matches[3]))
		} else {
			fmt.Println("Failed to parse arguments")
		}
		return
	} else {
		// use as a host
		sessionID := os.Getenv("TRANSFER_SESSIONID")
		if sessionID != "" {
			fmt.Printf("This terminal is already being shared at: %s\n", transferringfile.GetClientURL(*server, sessionID))
			return
		}
		if args[0] == "send" {
			filePath = args[1]
		}

		// TODO: Host need to register a file to server.
		tf := transferringfile.New(*noTurn)
		tf.Start(*server, filePath)
		return
	}
}
