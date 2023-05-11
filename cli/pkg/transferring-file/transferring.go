package transferringfile

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	// "webrtc/cli/pkg/message"
	// "webrtc/cli/pkg/pty"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/minhnh/transfer-file/cli/pkg/message"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	transferChannel *webrtc.DataChannel

	configChannel *webrtc.DataChannel

	conn *webrtc.PeerConnection

	authenticated bool
}

type Transfer struct {
	// pty *pty.Pty

	wsConn *Websocket
	lock   sync.RWMutex

	clients  map[string]*Client
	noTurn   bool
	passcode string
}

func New(noTurn bool) *Transfer {
	return &Transfer{
		clients: make(map[string]*Client),
		noTurn:  noTurn,
	}
}

func (tf *Transfer) Start(server string, filePath string) error {
	// Create a pty to fake the terminal session
	sessionID := uuid.NewString()
	log.Printf("New session: %s", sessionID)
	// envVars := []string{fmt.Sprintf("%s=%s", "TRANSFER_SESSIONID", sessionID)}
	// tf.pty.StartDefaultShell(envVars)
	fmt.Println("zzxzz", filePath)
	// Set passcode
	fmt.Printf("Set passcode (enter to disable passcode): ")
	for {
		passcode, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		passcode = strings.TrimSpace(passcode)
		// enter to set no passcode
		if len(passcode) == 0 {
			break
		}

		err := tf.SetPasscode(passcode)
		if err != nil {
			fmt.Printf("%s\n", err)
			fmt.Printf("Set passcode (enter to disable passcode): ")
		} else {
			break
		}
	}

	fmt.Printf("Sharing at: %s\n", GetClientURL(server, sessionID))
	fmt.Println("Type 'exit' or press 'Ctrl-D' to exit")
	// tf.pty.MakeRaw()
	defer tf.Stop("Bye!")

	wsURL := GetWSURL(server, sessionID)
	// log.Println("Connecting to:%s", wsURL)
	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to signaling server: %s", err)
		tf.Stop("Failed to connect to signaling server")
		return err
	}
	tf.wsConn = wsConn
	go wsConn.Start()

	// send a ping message to keep websocket alive, doesn't expect to receive anything
	// This messages is expected to be broadcast to all client's connections so it keeps them alive too
	go func() {
		for range time.Tick(5 * time.Second) {
			payload := message.Wrapper{
				Type: "Ping",
				Data: []byte{},
			}
			tf.writeWebsocket(payload)
		}
	}()

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// use io.Copy to simultaneously upload and download
	_, err = io.Copy(tf, file)
	if err != nil {
		panic(err)
	}

	wsConn.SetPingHandler(func(appData string) error {
		return wsConn.WriteControl(websocket.PongMessage, []byte{}, time.Time{})
	})

	wsConn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket connection closed with code %d :%s", code, text)
		tf.Stop("WebSocket connection to server is closed")
		return nil
	})

	requirePasscodeMsg := message.Wrapper{
		Type: "RequirePasscode",
	}
	if !tf.isRequirePasscode() {
		requirePasscodeMsg.Type = "NoPasscode"
	}
	tf.writeWebsocket(requirePasscodeMsg)

	go tf.startHandleWsMessages()

	// tf.pty.Wait() // Blocking until user exit
	select {}
}

func (tf *Transfer) Stop(msg string) {
	if tf.wsConn != nil {
		tf.wsConn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		tf.wsConn.Close()
		tf.wsConn = nil
	}

	for _, client := range tf.clients {
		client.conn.Close()
	}
}

// shortcut to write to websocket connection
func (tf *Transfer) writeWebsocket(msg message.Wrapper) error {
	msg.From = "host"
	if tf.wsConn == nil {
		return fmt.Errorf("Websocket not connected")
	}
	tf.wsConn.Out <- msg
	return nil
}

// Blocking call to connect to a websocket server for signaling
func (tf *Transfer) startHandleWsMessages() error {
	log.Printf("zzzzzz")
	if tf.wsConn == nil {
		log.Printf("Websocket connection not initialized")
		return fmt.Errorf("Websocket connection not initialized")
	}

	for {
		log.Print("zcxzxczxc123123123123")
		msg, ok := <-tf.wsConn.In
		// log.Printf("zzzzz", msg)
		if !ok {
			log.Printf("Failed to read websocket message")
			return fmt.Errorf("Failed to read message from websocket")
		}

		// skip messages that are not send to the host
		if msg.To != "host" {
			log.Printf("Skip message :%s", msg)
			continue
		}

		err := tf.handleWebSocketMessage(msg)
		if err != nil {
			log.Printf("Failed to handle message: %v, with error: %s", msg, err)
			continue
		}
	}
}

func (tf *Transfer) handleWebSocketMessage(msg message.Wrapper) error {
	log.Printf("zczxczx")
	// var client *Client
	if msg.Type == "Connect" {
		// _, err := tf.newClient(msg.From)
		log.Printf("New client with ID: %s", msg.From)
		// if err != nil {
		// 	return fmt.Errorf("Failed to create client: %s", err)
		// }
	}
	return nil
}

func (tf *Transfer) isRequirePasscode() bool {
	return len(tf.passcode) > 0
}

func (tf *Transfer) SetPasscode(passcode string) error {
	if len(passcode) >= 6 {
		tf.passcode = passcode
		return nil
	} else {
		return fmt.Errorf("Passcode must be more than 6 characters")
	}
}

// Write method to forward terminal changes over webrtc
func (tf *Transfer) Write(data []byte) (int, error) {
	tf.lock.RLock()
	defer tf.lock.RUnlock()

	for ID, client := range tf.clients {
		//go func(ID string, client *Client) {
		if client.transferChannel != nil {
			err := client.transferChannel.Send(data)
			if err != nil {
				log.Printf("Failed to send config to client: %s", ID)
			}
		}
		//}(ID, client)
	}

	return len(data), nil
}
