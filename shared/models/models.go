package models

import "time"

type DeviceStatus string

const (
	DeviceConnected DeviceStatus = "connected"
	DeviceOffline   DeviceStatus = "offline"
)

type DeviceIdentity struct {
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

type PairingPayload struct {
	Mode   string `json:"mode"`
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Token  string `json:"token,omitempty"`
	Relay  string `json:"relay,omitempty"`
	Public string `json:"public,omitempty"`
	Room   string `json:"room,omitempty"`
}

type TrustedDevice struct {
	DeviceIdentity
	Status   DeviceStatus `json:"status"`
	LastSeen time.Time    `json:"lastSeen"`
}
