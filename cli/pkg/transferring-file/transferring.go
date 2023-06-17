package transferringfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	// "webrtc/cli/pkg/message"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/k0kubun/go-ansi"
	"github.com/minhnh/transfer-file/cli/pkg/message"
	"github.com/minhnh/transfer-file/internal/cfg"
	"github.com/pion/webrtc/v3"
	"github.com/schollz/progressbar/v3"
)

type Client struct {
	transferChannel *webrtc.DataChannel

	conn *webrtc.PeerConnection

	authenticated bool
}

type Transfer struct {
	wsConn *Websocket
	lock   sync.RWMutex

	clients  map[string]*Client
	noTurn   bool
	passcode string

	file *os.File

	// for progressbar
	bar    *progressbar.ProgressBar
	doneCh chan struct{}
}

type FileInformation struct {
	Name    string
	Size    int64
	Content []byte
}

type MesageChannel struct {
	Type string
	Data FileInformation
}

func New(noTurn bool) *Transfer {
	return &Transfer{
		clients: make(map[string]*Client),
		noTurn:  noTurn,
	}
}

func (tf *Transfer) Start(server string, filePath string) error {
	sessionID := uuid.NewString()
	log.Printf("New session: %s", sessionID)

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
	defer tf.Stop("Bye!")

	wsURL := GetWSURL(server, sessionID)
	log.Println("Connecting to:%s", wsURL)
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
	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	tf.file = file
	tf.writeWebsocket(requirePasscodeMsg)

	go tf.startHandleWsMessages()

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
	msg.From = cfg.TRANSFER_WEBSOCKET_HOST_ID
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
		if msg.To != cfg.TRANSFER_WEBSOCKET_HOST_ID {
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
	var client *Client
	if msg.Type == message.TCConnect {
		clientVersion := msg.Data.(string)

		if clientVersion != cfg.SUPPORTED_VERSION {
			tf.writeWebsocket(message.Wrapper{Type: message.TCUnsupportedVersion, Data: cfg.SUPPORTED_VERSION, To: msg.From})
			return fmt.Errorf("Client is running unsupported version :%s", clientVersion)
		}

		_, err := tf.newClient(msg.From)
		log.Printf("New client with ID: %s", msg.From)
		if err != nil {
			return fmt.Errorf("Failed to create client: %s", err)
		}

		msg := message.Wrapper{
			To: msg.From,
		}

		if tf.isRequirePasscode() {
			msg.Type = message.TCRequirePasscode
		} else {
			fileInfo, _ := tf.file.Stat()
			msg.Type = message.TCNoPasscode
			msg.Data = map[string]interface{}{
				"fileName": fileInfo.Name(),
				"fileSize": fileInfo.Size(),
			}
		}

		tf.writeWebsocket(msg)
		return nil
	}
	client, ok := tf.clients[msg.From]

	if !ok {
		return fmt.Errorf("Client with ID: %s not found", msg.From)
	}

	switch msgType := msg.Type; msgType {
	// offer
	case message.TRTCOffer:

		if tf.isRequirePasscode() && !client.authenticated {
			return fmt.Errorf("Unauthenticated client")
		}

		offer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &offer); err != nil {
			return err
		}
		log.Printf("Get an offer: %v", (string(msg.Data.(string))))

		if err := client.conn.SetRemoteDescription(offer); err != nil {
			return fmt.Errorf("Failed to set remote description: %s", err)
		}

		// send back SDP answer and set it as local description
		answer, err := client.conn.CreateAnswer(nil)
		if err != nil {
			return fmt.Errorf("Failed to create offfer: %s", err)
		}

		if err := client.conn.SetLocalDescription(answer); err != nil {
			return fmt.Errorf("Failed to set local description: %s", err)
		}
		answerByte, _ := json.Marshal(answer)
		payload := message.Wrapper{
			Type: message.TRTCAnswer,
			Data: string(answerByte),
			To:   msg.From,
		}
		tf.writeWebsocket(payload)

	case message.TRTCCandidate:
		if tf.isRequirePasscode() && !client.authenticated {
			return fmt.Errorf("Unauthenticated client")
		}
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &candidate); err != nil {
			return fmt.Errorf("Failed to unmarshall icecandidate: %s", err)
		}

		if err := client.conn.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("Failed to add ice candidate: %s", err)
		}

	case message.TCPasscode:
		passcode := msg.Data.(string)
		fileInfo, _ := tf.file.Stat()
		resp := message.Wrapper{
			To: msg.From,
			Data: map[string]interface{}{
				"fileName": fileInfo.Name(),
				"fileSize": fileInfo.Size(),
			},
		}

		if tf.isAuthenticated(passcode) {
			client.authenticated = true
			resp.Type = message.TCAuthenticated
		} else {
			resp.Type = message.TCUnauthenticated
		}
		tf.writeWebsocket(resp)

	default:
		return fmt.Errorf("Not implemented to handle message type: %s", msg.Type)
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

func (tf *Transfer) removeClient(ID string) {
	if client, ok := tf.clients[ID]; ok {
		tf.lock.Lock()
		defer tf.lock.Unlock()

		if client.transferChannel != nil {
			client.transferChannel.Close()
			client.transferChannel = nil
		}

		if client.conn != nil {
			client.conn.Close()
		}

		delete(tf.clients, ID)
	}
}

func (tf *Transfer) newClient(ID string) (*Client, error) {
	// Initiate peer connection
	ICEServers := cfg.TRANSFER_ICE_SERVER_STUNS
	// if !ts.noTurn {
	// 	ICEServers = append(ICEServers, cfg.TERMISHARE_ICE_SERVER_TURNS...)
	// }

	var config = webrtc.Configuration{
		ICEServers: ICEServers,
	}

	client := &Client{authenticated: false}

	tf.lock.Lock()
	tf.clients[ID] = client
	tf.lock.Unlock()

	peerConn, err := webrtc.NewPeerConnection(config)

	if err != nil {
		fmt.Printf("Failed to create peer connection: %s", err)
		return nil, err
	}
	client.conn = peerConn

	// Create channel that is blocked until ICE Gathering is complete
	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
		switch s {
		// case webrtc.PeerConnectionStateConnected:
		case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateDisconnected:
			log.Printf("Removing client: %s", ID)
			tf.removeClient(ID)
		}
	})

	peerConn.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", d.Label(), d.ID())
		// Register channel opening handling
		d.OnOpen(func() {
			switch label := d.Label(); label {

			case cfg.TRANSFER_WEBRTC_DATA_CHANNEL:
				//Send file after established peer to peer
				fileInfo, err := tf.file.Stat()
				if err != nil {
					log.Printf("Failed to get file information: %w", err)
					break
				}

				tf.TfinitiateBar(int(fileInfo.Size()))
				// Send file information to client.
				fileInfoJSON, _ := json.Marshal(MesageChannel{
					Type: "Send",
					Data: FileInformation{
						Name: fileInfo.Name(),
						Size: fileInfo.Size(),
					},
				})

				d.Send(fileInfoJSON)
				// fmt.Println("Sending '%s' (%s)", fileInfo.Name(), ByteCountDecimal(int64(fileInfo.Size())))
				d.OnMessage(func(msg webrtc.DataChannelMessage) {
					webrtcMessage := MesageChannel{}
					err := json.Unmarshal(msg.Data, &webrtcMessage)
					if err != nil {
						log.Printf("Error unmarshaling file information:", err)
						return
					}
					switch webrtcMessage.Type {
					case "Received":
						// We can't reading the entire file into memory. Therefore, loop to read a portion of the file, send it,
						// and continue until the entire file is transmitted.
						buffer := make([]byte, 4096)
						for {
							bytesRead, err := tf.file.Read(buffer)
							if err != nil && err != io.EOF {
								log.Printf("Failed to read file %w", err)
								break
							}
							if bytesRead > 0 {
								dataByte, errs := json.Marshal(MesageChannel{
									Type: "Content",
									Data: FileInformation{
										Name:    fileInfo.Name(),
										Size:    fileInfo.Size(),
										Content: buffer[:bytesRead],
									},
								})
								if errs != nil {
									log.Printf("failed to marshal file information: %w", errs)
									break
								}
								err = d.Send(dataByte)
								tf.bar.Add(bytesRead)
								if err != nil {
									log.Printf("failed to send file data: %w", err)
									break
								}

							}
						}
					}
				})
				tf.clients[ID].transferChannel = d
			default:
				log.Printf("Unhandled data channel with label: %s", d.Label())
			}
		})
	})

	peerConn.OnICECandidate(func(ice *webrtc.ICECandidate) {
		if ice == nil {
			return
		}

		candidate, err := json.Marshal(ice.ToJSON())
		if err != nil {
			log.Printf("Failed to decode ice candidate: %s", err)
			return
		}

		msg := message.Wrapper{
			Type: message.TRTCCandidate,
			Data: string(candidate),
			To:   ID,
		}

		tf.writeWebsocket(msg)
	})
	return client, nil
}

func (tf *Transfer) isAuthenticated(passcode string) bool {
	return passcode == tf.passcode
}

func (tf *Transfer) TfinitiateBar(sizeFile int) {
	tf.bar = progressbar.NewOptions(sizeFile,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetDescription("Writing moshable file..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			tf.doneCh <- struct{}{}
		}))
}
