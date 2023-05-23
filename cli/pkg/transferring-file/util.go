package transferringfile

import (
	"fmt"
	"net/url"
	"strings"
)

func GetClientURL(client string, sessionID string) string {
	return fmt.Sprintf("%s/%s", client, sessionID)
}

func GetWSURL(server string, sessionID string) string {
	// Initiate websocket connection for signaling
	scheme := "ws"
	if strings.HasPrefix(server, "https") || strings.HasPrefix(server, "wss") {
		scheme = "wss"
	}
	host := strings.Replace(strings.Replace(server, "http://", "", 1), "https://", "", 1)
	url := url.URL{Scheme: scheme, Host: host, Path: fmt.Sprintf("/%s", sessionID)}
	return url.String()
}

// ByteCountDecimal converts bytes to human readable byte string
func ByteCountDecimal(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
