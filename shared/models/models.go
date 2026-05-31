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
}

type PairingPayload struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Token string `json:"token"`
}

type TrustedDevice struct {
	DeviceIdentity
	Status   DeviceStatus `json:"status"`
	LastSeen time.Time    `json:"lastSeen"`
}
