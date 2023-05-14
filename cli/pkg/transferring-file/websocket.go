package transferringfile

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/minhnh/transfer-file/cli/pkg/message"
)

type Websocket struct {
	*websocket.Conn
	In             chan message.Wrapper
	Out            chan message.Wrapper
	lastActiveTime time.Time
	active         bool
}

func NewWebSocketConnection(url string) (*Websocket, error) {
	log.Println(url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &Websocket{
		Conn:   conn,
		In:     make(chan message.Wrapper, 256),
		Out:    make(chan message.Wrapper, 256),
		active: true,
	}, nil
}

// blocking method that start receive and send websocket message
func (ws *Websocket) Start() {

	// Receive message coroutine
	go func() {
		for {
			msg, ok := <-ws.Out
			ws.lastActiveTime = time.Now()
			if ok {
				err := ws.WriteJSON(msg)
				if err != nil {
					log.Printf("Failed to send mesage : %s", err)
					ws.Stop()
					break
				}
			} else {
				log.Printf("Failed to get message from channel")
				ws.Stop()
				break
			}
		}
	}()

	// Send message coroutine
	for {
		msg := message.Wrapper{}
		fmt.Printf("fsdfsdfsdfsdf34 %s", msg)
		err := ws.ReadJSON(&msg)
		if err == nil {
			ws.In <- msg // Will be handled in Room
		} else {
			log.Printf("Failed to read message. Closing connection: %s", err)
			ws.Stop()
			break
		}
	}
	log.Printf("Out websocket")
}

// Gracefully close websocket connection
func (ws *Websocket) Stop() {
	if ws.active {
		ws.active = false
		log.Printf("Closing client")
		ws.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		time.Sleep(1 * time.Second) // give client sometimes to receive the control message
		close(ws.In)
		close(ws.Out)
		ws.Close()
	}
}
