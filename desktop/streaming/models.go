package streaming

import "time"

type Metrics struct {
	FPS        int       `json:"fps"`
	Codec      string    `json:"codec"`
	Transport  string    `json:"transport"`
	LatencyMS  int       `json:"latencyMs"`
	Frames     uint64    `json:"frames"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Resolution string    `json:"resolution"`
}

// StreamState tells the UI the truth about the video pipeline instead of a
// generic spinner: idle → connected (device paired) → starting (stream.start
// received) → streaming (fresh frames).
type StreamState struct {
	State         string `json:"state"` // idle | connected | starting | streaming
	ActiveDevice  string `json:"activeDevice"`
	LastFrameAgoMs int64 `json:"lastFrameAgeMs"`
}

type FrameEvent struct {
	Codec        string `json:"codec"`
	Data         string `json:"data"`
	KeyFrame     bool   `json:"keyFrame"`
	TimestampNS  int64  `json:"timestampNs"`
	ReceivedAtNS int64  `json:"receivedAtNs"`
}

type Envelope struct {
	Type string `json:"type"`
}

type PairVerifyMessage struct {
	Type         string                 `json:"type"`
	Token        string                 `json:"token"`
	Device       DeviceHandshakePayload `json:"device"`
	Capabilities map[string]any         `json:"capabilities"`
}

type DeviceHandshakePayload struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Model          string `json:"model"`
	Manufacturer   string `json:"manufacturer"`
	Platform       string `json:"platform"`
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	AndroidVersion string `json:"androidVersion"`
}

type ReconnectMessage struct {
	Type        string `json:"type"`
	DeviceID    string `json:"deviceId"`
	TrustSecret string `json:"trustSecret"`
}

type StreamStartMessage struct {
	Type      string `json:"type"`
	Codec     string `json:"codec"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	TargetFPS int    `json:"targetFps"`
	DeviceID  string `json:"deviceId"`
}
