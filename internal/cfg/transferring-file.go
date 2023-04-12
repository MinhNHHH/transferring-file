package cfg

import (
	"github.com/pion/webrtc/v3"
)

var TERMISHARE_ICE_SERVER_STUNS = []webrtc.ICEServer{{URLs: []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"}}}

var TERMISHARE_ICE_SERVER_TURNS = []webrtc.ICEServer{{
	URLs:       []string{"turn:104.237.1.191:3478"},
	Username:   "termishare",
	Credential: "termishareisfun"}}
