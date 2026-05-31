package pairing

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"voixpe3per/desktop/network"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	ModeLAN   = "lan"
	ModeRelay = "relay"

	defaultPairingPageURL = "https://voixxpe3per.vercel.app/pair"
)

type Service struct {
	lan   *network.LANDetector
	store Store
	port  int

	mu      sync.RWMutex
	session SessionSnapshot
	devices map[string]trustedDevice
}

func NewService(lan *network.LANDetector, store Store, port int) *Service {
	service := &Service{
		lan:     lan,
		store:   store,
		port:    port,
		devices: make(map[string]trustedDevice),
	}
	_ = service.load()
	return service
}

func (s *Service) StartSession() error {
	relayURL := strings.TrimSpace(os.Getenv("VOIXPE3PER_RELAY_URL"))
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("VOIXPE3PER_PAIRING_MODE")))
	if mode == "" && relayURL != "" {
		mode = ModeRelay
	}
	if mode == "" {
		mode = ModeRelay
	}

	if mode == ModeRelay {
		return s.startRelaySession(relayURL)
	}

	host, err := s.lan.LocalIPv4()
	if err != nil {
		host = "127.0.0.1"
	}

	token, err := randomToken(32)
	if err != nil {
		return err
	}

	payload := PairingPayload{
		Mode:  ModeLAN,
		Host:  host,
		Port:  s.port,
		Token: token,
	}
	qrTarget := pairingURL(payload)
	png, err := qrcode.Encode(qrTarget, qrcode.Medium, 384)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = SessionSnapshot{
		Host:      host,
		Port:      s.port,
		Token:     token,
		Mode:      ModeLAN,
		QRDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		Status:    "Waiting for device...",
	}
	return nil
}

func (s *Service) startRelaySession(relayURL string) error {
	if relayURL == "" {
		relayURL = "ws://127.0.0.1:8090/ws"
	}

	room, err := randomRoom()
	if err != nil {
		return err
	}

	payload := PairingPayload{
		Mode:  ModeRelay,
		Relay: relayURL,
		Room:  room,
	}
	qrTarget := pairingURL(payload)
	png, err := qrcode.Encode(qrTarget, qrcode.Medium, 384)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = SessionSnapshot{
		Mode:      ModeRelay,
		RelayURL:  relayURL,
		Room:      room,
		QRDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		Status:    "Waiting through relay...",
	}
	return nil
}

func (s *Service) Snapshot() SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.session
}

func (s *Service) Devices() []DeviceView {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]DeviceView, 0, len(s.devices))
	for _, device := range s.devices {
		devices = append(devices, toView(device))
	}
	return devices
}

func (s *Service) VerifyPairing(token string, handshake DeviceHandshake) (PairSuccess, error) {
	s.mu.RLock()
	currentToken := s.session.Token
	mode := s.session.Mode
	s.mu.RUnlock()

	if mode != ModeRelay && (token == "" || token != currentToken) {
		return PairSuccess{}, fmt.Errorf("invalid pairing token")
	}

	deviceID := strings.TrimSpace(handshake.ID)
	if deviceID == "" {
		generated, err := randomToken(16)
		if err != nil {
			return PairSuccess{}, err
		}
		deviceID = platform(handshake) + "-" + generated
	}

	trustSecret, err := randomToken(48)
	if err != nil {
		return PairSuccess{}, err
	}

	device := trustedDevice{
		ID:              deviceID,
		Name:            fallback(handshake.Name, handshake.Model, "Mobile Device"),
		Model:           fallback(handshake.Model, "Unknown Model"),
		Manufacturer:    fallback(handshake.Manufacturer, "Unknown"),
		Platform:        platform(handshake),
		OSName:          osName(handshake),
		OSVersion:       osVersion(handshake),
		AndroidVersion:  fallback(handshake.AndroidVersion, handshake.OSVersion, "unknown"),
		Status:          DeviceConnected,
		LastSeen:        time.Now().UTC(),
		TrustSecretHash: hashSecret(trustSecret),
	}

	s.mu.Lock()
	s.devices[deviceID] = device
	s.session.Status = "Connected"
	s.mu.Unlock()

	if err := s.persist(); err != nil {
		return PairSuccess{}, err
	}

	return PairSuccess{
		Type:        "pair.success",
		DeviceID:    deviceID,
		TrustSecret: trustSecret,
		Device:      toView(device),
	}, nil
}

func (s *Service) VerifyReconnect(deviceID, trustSecret string) (DeviceView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return DeviceView{}, fmt.Errorf("device is not trusted")
	}
	if device.TrustSecretHash != hashSecret(trustSecret) {
		return DeviceView{}, fmt.Errorf("trust secret mismatch")
	}
	device.Status = DeviceConnected
	device.LastSeen = time.Now().UTC()
	s.devices[deviceID] = device
	s.session.Status = "Connected"
	go func() { _ = s.persist() }()
	return toView(device), nil
}

func (s *Service) MarkDeviceOffline(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok := s.devices[deviceID]
	if !ok {
		return
	}
	device.Status = DeviceOffline
	device.LastSeen = time.Now().UTC()
	s.devices[deviceID] = device
	go func() { _ = s.persist() }()
}

func (s *Service) ForgetDevice(deviceID string) error {
	s.mu.Lock()
	delete(s.devices, deviceID)
	s.mu.Unlock()
	return s.persist()
}

func (s *Service) load() error {
	devices, err := s.store.Load()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, device := range devices {
		device.Status = DeviceOffline
		s.devices[device.ID] = device
	}
	return nil
}

func (s *Service) persist() error {
	s.mu.RLock()
	devices := make([]trustedDevice, 0, len(s.devices))
	for _, device := range s.devices {
		device.Status = DeviceOffline
		devices = append(devices, device)
	}
	s.mu.RUnlock()
	return s.store.Save(devices)
}

func toView(device trustedDevice) DeviceView {
	return DeviceView{
		ID:             device.ID,
		Name:           device.Name,
		Model:          device.Model,
		Manufacturer:   device.Manufacturer,
		Platform:       fallback(device.Platform, "android"),
		OSName:         fallback(device.OSName, "Android"),
		OSVersion:      fallback(device.OSVersion, device.AndroidVersion, "unknown"),
		AndroidVersion: device.AndroidVersion,
		Status:         device.Status,
		LastSeen:       device.LastSeen,
	}
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func pairingURL(payload PairingPayload) string {
	pageURL := strings.TrimSpace(os.Getenv("VOIXPE3PER_PAIRING_PAGE_URL"))
	if pageURL == "" {
		pageURL = defaultPairingPageURL
	}

	parsed, err := url.Parse(pageURL)
	if err != nil || parsed.Scheme == "" {
		raw, _ := json.Marshal(payload)
		return string(raw)
	}

	query := parsed.Query()
	query.Set("mode", payload.Mode)
	if payload.Relay != "" {
		query.Set("relay", payload.Relay)
	}
	if payload.Room != "" {
		query.Set("room", payload.Room)
	}
	if payload.Host != "" {
		query.Set("host", payload.Host)
	}
	if payload.Port > 0 {
		query.Set("port", fmt.Sprintf("%d", payload.Port))
	}
	if payload.Token != "" {
		query.Set("token", payload.Token)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func randomRoom() (string, error) {
	token, err := randomToken(9)
	if err != nil {
		return "", err
	}
	clean := strings.ToUpper(strings.NewReplacer("-", "", "_", "").Replace(token))
	if len(clean) > 12 {
		clean = clean[:12]
	}
	return clean, nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func platform(handshake DeviceHandshake) string {
	value := strings.ToLower(strings.TrimSpace(handshake.Platform))
	if value != "" {
		return value
	}
	if strings.Contains(strings.ToLower(handshake.Manufacturer), "apple") {
		return "ios"
	}
	return "android"
}

func osName(handshake DeviceHandshake) string {
	if strings.TrimSpace(handshake.OSName) != "" {
		return handshake.OSName
	}
	if platform(handshake) == "ios" {
		return "iOS"
	}
	return "Android"
}

func osVersion(handshake DeviceHandshake) string {
	return fallback(handshake.OSVersion, handshake.AndroidVersion, "unknown")
}

func fallback(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
