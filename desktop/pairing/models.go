package pairing

import "time"

type DeviceStatus string

const (
	DeviceConnected DeviceStatus = "connected"
	DeviceOffline   DeviceStatus = "offline"
)

type PairingPayload struct {
	Mode   string `json:"mode"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Token  string `json:"token,omitempty"`
	Relay  string `json:"relay,omitempty"`
	Public string `json:"public,omitempty"`
	Room   string `json:"room,omitempty"`
}

type SessionSnapshot struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Token     string `json:"token"`
	Mode      string `json:"mode"`
	RelayURL  string `json:"relayUrl"`
	Room      string `json:"room"`
	QRDataURL string `json:"qrDataUrl"`
	Status    string `json:"status"`
}

type DeviceView struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Model          string       `json:"model"`
	Manufacturer   string       `json:"manufacturer"`
	Platform       string       `json:"platform"`
	OSName         string       `json:"osName"`
	OSVersion      string       `json:"osVersion"`
	AndroidVersion string       `json:"androidVersion"`
	StreamCapable  bool         `json:"streamCapable"`
	Status         DeviceStatus `json:"status"`
	LastSeen       time.Time    `json:"lastSeen"`
}

type trustedDevice struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Model           string       `json:"model"`
	Manufacturer    string       `json:"manufacturer"`
	Platform        string       `json:"platform"`
	OSName          string       `json:"osName"`
	OSVersion       string       `json:"osVersion"`
	AndroidVersion  string       `json:"androidVersion"`
	StreamCapable   bool         `json:"streamCapable"`
	Status          DeviceStatus `json:"status"`
	LastSeen        time.Time    `json:"lastSeen"`
	TrustSecretHash string       `json:"trustSecretHash"`
}

type DeviceHandshake struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Model          string `json:"model"`
	Manufacturer   string `json:"manufacturer"`
	Platform       string `json:"platform"`
	OSName         string `json:"osName"`
	OSVersion      string `json:"osVersion"`
	AndroidVersion string `json:"androidVersion"`
	StreamCapable  bool   `json:"streamCapable"`
}

type PairSuccess struct {
	Type        string     `json:"type"`
	DeviceID    string     `json:"deviceId"`
	TrustSecret string     `json:"trustSecret"`
	Device      DeviceView `json:"device"`
}
