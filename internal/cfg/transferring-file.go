package cfg

import (
	"github.com/pion/webrtc/v3"
)

const (
	TRANSFER_WEBSOCKET_CHANNEL_SIZE = 256 // TRANSFER channel buffer size for websocket

	TRANSFER_ENVKEY_SESSIONID      = "TRANSFER_SESSIONID" // name of env var to keep sessionid value
	TRANSFER_WEBRTC_DATA_CHANNEL   = "transfer"           // lable name of webrtc data channel to exchange byte data
	TRANSFER_WEBRTC_CONFIG_CHANNEL = "config"             // lable name of webrtc config channel to exchange config
	TRANSFER_WEBSOCKET_HOST_ID     = "host"               // ID of message sent by the host

	TRANSFER_VERSION  = "0.0.4"
	SUPPORTED_VERSION = "0.0.4" // the oldest TRANSFER version of client that the host could support
)

var TRANSFER_ICE_SERVER_STUNS = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}}}

// var TRANSFER_ICE_SERVER_TURNS = []webrtc.ICEServer{{
// 	URLs:       []string{"turn:104.237.1.191:3478"},
// 	Username:   "TRANSFER",
// 	Credential: "termishareisfun"}}
