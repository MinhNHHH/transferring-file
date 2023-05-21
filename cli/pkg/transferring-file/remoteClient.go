package transferringfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/minhnh/transfer-file/cli/pkg/message"
	"github.com/minhnh/transfer-file/internal/cfg"
	"github.com/pion/webrtc/v3"
)

type RemoteClient struct {
	clientID string

	// use Client struct
	// for transferring terminal changes
	dataChannel *webrtc.DataChannel
	peerConn    *webrtc.PeerConnection
	wsConn      *Websocket

	connected bool
	done      chan bool
	answer    string
	file      []byte
}
type TCAuthenticatedData struct {
	fileName string
	fileSize int64
}

func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		clientID:  uuid.NewString(),
		connected: false,
		done:      make(chan bool),
	}
}

// When coonect to link, connection need to download file.
func (rc *RemoteClient) Connect(server string, sessionID string) {
	log.Printf("Start")
	wsURL := GetWSURL(server, sessionID)
	fmt.Printf("Connecting to: %s\n", wsURL)

	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to singaling server : %s", err)
	}
	go wsConn.Start()

	// Initiate peer to peer
	iceServers := cfg.TRANSFER_ICE_SERVER_STUNS

	config := webrtc.Configuration{
		ICEServers:   iceServers,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}

	peerConn, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connetion : %s", err)
	}

	rc.peerConn = peerConn

	peerConn.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		log.Printf("Peer connection state has changed: %s", s.String())
		switch s {
		case webrtc.PeerConnectionStateClosed:
		case webrtc.PeerConnectionStateDisconnected:
		case webrtc.PeerConnectionStateFailed:
			rc.peerConn.Close()
			log.Printf("Disconnected")
		}
	})

	dataChannel, err := peerConn.CreateDataChannel(cfg.TRANSFER_WEBRTC_DATA_CHANNEL, nil)
	rc.dataChannel = dataChannel

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		downloadFile, err := os.Create("file-to-download.txt")
		if err != nil {
			log.Printf("Error create new file", err)
		}
		defer downloadFile.Close()
		_, err = downloadFile.Write(msg.Data)
		if err != nil {
			log.Printf("Error download file", err)
		}
		fmt.Printf("Download Succes")
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
		}

		rc.writeWebsocket(msg)
	})

	rc.wsConn = wsConn
	rc.writeWebsocket(message.Wrapper{
		Type: message.TCConnect,
		Data: cfg.SUPPORTED_VERSION})

	for {
		msg, ok := <-rc.wsConn.In
		if !ok {
			log.Printf("Failed to read websocket message")
			break
		}

		// only read message sent from the host
		if msg.From != cfg.TRANSFER_WEBSOCKET_HOST_ID {
			log.Printf("Skip message :%v", msg)
		}

		err := rc.handleWebSocketMessage(msg)
		if err != nil {
			log.Printf("Failed to handle message: %v, with error: %s", msg, err)
			break
		}
	}
	<-rc.done
	return
}

func (rc *RemoteClient) writeWebsocket(msg message.Wrapper) error {
	msg.To = cfg.TRANSFER_WEBSOCKET_HOST_ID
	msg.From = rc.clientID
	if rc.wsConn == nil {
		return fmt.Errorf("Websocket not connected")
	}
	rc.wsConn.Out <- msg
	return nil
}

func (rc *RemoteClient) Stop(msg string) {
	log.Printf("Stop: %s", msg)

	if rc.wsConn != nil {
		rc.wsConn.WriteControl(websocket.CloseMessage, []byte{}, time.Time{})
		rc.wsConn.Close()
		rc.wsConn = nil
	}

	if rc.peerConn != nil {
		rc.peerConn.Close()
		rc.peerConn = nil
	}

	fmt.Println(msg)
	rc.done <- true
	return
}

func (rc *RemoteClient) handleWebSocketMessage(msg message.Wrapper) error {
	switch msgType := msg.Type; msgType {

	case message.TCUnsupportedVersion:
		rc.Stop(fmt.Sprintf("The host require version:"))

	case message.TCUnauthenticated:
		fmt.Printf("Incorrect passcode!\n")
		fallthrough

	case message.TCRequirePasscode:
		fmt.Printf("Passcode: ")
		passcode, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		passcode = strings.TrimSpace(passcode)
		resp := message.Wrapper{
			Type: message.TCPasscode,
			Data: passcode,
		}
		rc.writeWebsocket(resp)

	case message.TCNoPasscode, message.TCAuthenticated:

		data := msg.Data.(map[string]interface{})
		fmt.Printf("Accept '%s' (%0.1f MB)? (y/n) ", data["fileName"], data["fileSize"])
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		answer = strings.TrimSpace(answer)

		rc.answer = answer
		err := rc.SetAnswer(answer)
		if err != nil {
			return err
		}

	case message.TRTCOffer:
		return fmt.Errorf("Remote client shouldn't receive Offer message")

	case message.TRTCAnswer:
		answer := webrtc.SessionDescription{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &answer); err != nil {
			return err
		}
		rc.peerConn.SetRemoteDescription(answer)

	case message.TRTCCandidate:
		candidate := webrtc.ICECandidateInit{}
		if err := json.Unmarshal([]byte(msg.Data.(string)), &candidate); err != nil {
			return fmt.Errorf("Failed to unmarshall icecandidate: %s", err)
		}

		if err := rc.peerConn.AddICECandidate(candidate); err != nil {
			return fmt.Errorf("Failed to add ice candidate: %s", err)
		}

	case message.TWSPing:
		return nil

	default:
		log.Printf("Unhandled message type: %s", msgType)
		return nil
	}

	return nil
}

func (rc *RemoteClient) sendOffer() {

	offer, err := rc.peerConn.CreateOffer(nil)
	if err != nil {
		log.Printf("Failed to create offer :%s", err)
		rc.Stop("Failed to connect to termishare session")
	}

	err = rc.peerConn.SetLocalDescription(offer)
	if err != nil {
		log.Printf("Failed to set local description: %s", err)
		rc.Stop("Failed to connect to termishare session")
	}
	offerByte, _ := json.Marshal(offer)
	payload := message.Wrapper{
		Type: message.TRTCOffer,
		Data: string(offerByte),
	}
	rc.writeWebsocket(payload)
}

func (rc *RemoteClient) SetAnswer(answer string) error {
	if answer == "y" {
		rc.answer = answer
		rc.connected = true
		rc.sendOffer()
		return nil
	} else {
		return fmt.Errorf("Answer should be y")
	}
}
