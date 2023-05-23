package message

type MType string
type Wrapper struct {
	Type MType
	Data interface{}
	From string
	To   string
}

const (
	// TODO refactor to make these message type as part of message
	// we probably only need RTC, Control, Connect types
	TRTCOffer     MType = "Offer"
	TRTCAnswer    MType = "Answer"
	TRTCCandidate MType = "Candidate"

	TTermWinsize MType = "Winsize" // Update winsize

	// Client can order the host to refresh the terminal
	// Used in case client resize and need to update the content to display correctly
	TTermRefresh MType = "Refresh"

	TWSPing MType = "Ping"

	// Whether or not a connection require a passcode
	// when connect, client will first send a connect message
	// server response with whether or not client needs to provide a passcode
	TCConnect         = "Connect"
	TCRequirePasscode = "RequirePasscode"
	TCNoPasscode      = "NoPasscode"
	TCSend            = "send"
	TCYes             = "y"
	TCNo              = "No"
	// message to wrap passcode
	TCPasscode = "Passcode"
	// connection's response
	TCAuthenticated   = "Authenticated"
	TCUnauthenticated = "Unauthenticated"

	TCUnsupportedVersion = "UnsupportedVersion"
)
