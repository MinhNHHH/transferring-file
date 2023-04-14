package transferringfile

import (
	"fmt"
	"log"
	"time"
	"webrtc/cli/pkg/message"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type Client struct {
	transferChannel *webrtc.DataChannel

	configChannel *webrtc.DataChannel

	conn *webrtc.PeerConnection

	authenticated bool
}

type Transfer struct {
	wsConn *Websocket

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

func (tf *Transfer) Start(server string) error {

	wsConn, err := NewWebSocketConnection(server)
	if err != nil {
		log.Printf("Failed to connect to signaling server: %s", err)
		tf.Stop("Failed to connect to signaling server")
		return err
	}
	tf.wsConn = wsConn
	go wsConn.Start()

	// send a ping message to keep websocket alive, doesn't expect to receive anything
	// This messages is expected to be broadcast to all client's connections so it keeps them alive too
	// go func() {
	// 	for range time.Tick(5 * time.Second) {
	// 		payload := message.Wrapper{
	// 			Type: "Ping",
	// 			Data: []byte{},
	// 		}
	// 		tf.writeWebsocket(payload)
	// 	}
	// }()

	wsConn.SetPingHandler(func(appData string) error {
		return wsConn.WriteControl(websocket.PongMessage, []byte{}, time.Time{})
	})

	wsConn.SetCloseHandler(func(code int, text string) error {
		log.Printf("WebSocket connection closed with code %d :%s", code, text)
		tf.Stop("WebSocket connection to server is closed")
		return nil
	})
	go tf.startHandleWsMessages()
	return nil
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
	fmt.Println(msg)
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
	if tf.wsConn == nil {
		log.Printf("Websocket connection not initialized")
		return fmt.Errorf("Websocket connection not initialized")
	}

	for {
		msg, ok := <-tf.wsConn.In
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
