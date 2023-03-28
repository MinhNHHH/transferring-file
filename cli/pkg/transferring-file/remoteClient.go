package transferringfile

import (
	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
)

type RemoteClient struct {
	clientID string

	// use Client struct
	// for transferring terminal changes
	datachannel   *webrtc.DataChannel
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
