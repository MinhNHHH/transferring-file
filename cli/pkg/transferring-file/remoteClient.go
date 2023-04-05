package transferringfile

import (
	"encoding/json"
	"log"
	"webrtc/cli/pkg/message"
	"webrtc/internal/cfg"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

type RemoteClient struct {
	clientID string

	// use Client struct
	// for transferring terminal changes
	dataChannel   *webrtc.DataChannel
	configChannel *webrtc.DataChannel
	peerConn      *webrtc.PeerConnection
	wsConn        *Websocket

	connected bool
	done      chan bool
}

func NewRemoteClient() *RemoteClient {
	return &RemoteClient{
		clientID:  uuid.NewString(),
		connected: false,
		done:      make(chan bool),
	}
}

func (rc *RemoteClient) Connect(server string, sessionID string) {
	log.Printf("Start")
	wsURL := "ws://localhost:8080/ws"

	wsConn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		log.Printf("Failed to connect to singaling server : %s", err)
	}
	go wsConn.Start()

	// Initiate peer to peer
	iceServers := cfg.TERMISHARE_ICE_SERVER_STUNS
	iceServers = append(iceServers, cfg.TERMISHARE_ICE_SERVER_TURNS...)

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

	configChannel, err := peerConn.CreateDataChannel("config", nil)
	dataChannel, err := peerConn.CreateDataChannel("transfer", nil)
	rc.configChannel = configChannel
	rc.dataChannel = dataChannel

	configChannel.OnMessage(func(webrtcMsg webrtc.DataChannelMessage) {
		msg := &message.Wrapper{}
		err := json.Unmarshal(webrtcMsg.Data, msg)
		if err != nil {
			log.Printf("Failed to read config message: %s", err)
			return
		}
		log.Printf("Config channel got msg: %v", msg)

	})
}
