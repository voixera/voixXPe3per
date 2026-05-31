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

	qrcode "github.com/skip2/go-qrcode"
)

const (
	ModeRelay  = "relay"
	ModeDirect = "direct"

	defaultPairingPageURL = "https://voixxpe3per.vercel.app/pair"
	defaultRelayURL       = "wss://voixpe3per-relay.onrender.com/ws"
)

type Service struct {
	store Store
	port  int

	mu      sync.RWMutex
	session SessionSnapshot
	devices map[string]trustedDevice
}

func NewService(store Store, port int) *Service {
	service := &Service{
		store:   store,
		port:    port,
		devices: make(map[string]trustedDevice),
	}
	_ = service.load()
	return service
}

func (s *Service) StartSession() error {
	publicURL := strings.TrimSpace(os.Getenv("VOIXPE3PER_PUBLIC_WS_URL"))
	relayURL := strings.TrimSpace(os.Getenv("VOIXPE3PER_RELAY_URL"))
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("VOIXPE3PER_PAIRING_MODE")))

	if mode == ModeDirect || (mode == "" && publicURL != "") {
		return s.startDirectSession(publicURL)
	}

	return s.startRelaySession(relayURL)
}

func (s *Service) startDirectSession(publicURL string) error {
	if publicURL == "" {
		return fmt.Errorf("VOIXPE3PER_PUBLIC_WS_URL is required for direct public pairing")
	}

	token, err := randomToken(32)
	if err != nil {
		return err
	}

	payload := PairingPayload{
		Mode:   ModeDirect,
		Token:  token,
		Relay:  publicURL,
		Public: publicURL,
	}
	qrTarget := pairingURL(payload)
	png, err := qrcode.Encode(qrTarget, qrcode.Medium, 384)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = SessionSnapshot{
		Token:     token,
		Mode:      ModeDirect,
		RelayURL:  publicURL,
		QRDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
		Status:    "Waiting through public WSS...",
	}
	return nil
}

func (s *Service) startRelaySession(relayURL string) error {
	if relayURL == "" {
		relayURL = defaultRelayURL
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
		StreamCapable:   handshake.StreamCapable,
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
		StreamCapable:  isStreamCapable(device),
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
	if payload.Public != "" {
		query.Set("public", payload.Public)
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

func isStreamCapable(device trustedDevice) bool {
	if device.StreamCapable {
		return true
	}
	return fallback(device.Platform, "android") != "web"
}

func fallback(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
