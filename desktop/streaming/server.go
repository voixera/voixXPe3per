package streaming

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"voixpe3per/desktop/pairing"

	"github.com/gorilla/websocket"
)

type Events struct {
	OnDeviceConnected    func(pairing.DeviceView)
	OnDeviceDisconnected func(deviceID string)
	OnFrame              func(FrameEvent)
	OnMetrics            func(Metrics)
}

type Server struct {
	addr    string
	pairing *pairing.Service

	httpServer *http.Server
	upgrader   websocket.Upgrader
	Events     Events

	mu              sync.RWMutex
	activeDeviceID  string
	connections     map[*websocket.Conn]string
	metrics         Metrics
	frameCounter    uint64
	done            chan struct{}
	requestKeyframe func()

	lastFrameNS   atomic.Int64 // unix nano of most recent frame
	streamStartNS atomic.Int64 // unix nano of last stream.start message
	loggedFrames  uint64       // throttled FRAME log counter
}

func NewServer(addr string, pairingService *pairing.Service) *Server {
	return &Server{
		addr:        addr,
		pairing:     pairingService,
		connections: make(map[*websocket.Conn]string),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		metrics: Metrics{
			Codec:     "H264",
			Transport: "Public WSS",
			UpdatedAt: time.Now().UTC(),
		},
	}
}

func (s *Server) Start() error {
	if s.httpServer != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("voiXPe3per desktop pairing server"))
	})
	mux.HandleFunc("/ws", s.handleWS)

	s.httpServer = &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		s.httpServer = nil
		return err
	}

	s.startMetrics()
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// The Wails layer reports startup failures through the first request path.
		}
	}()

	return nil
}

// StartRelay enables frame metrics without exposing a local HTTP/WebSocket port.
func (s *Server) StartRelay() {
	if s.done != nil {
		return
	}
	s.startMetrics()
}

func (s *Server) startMetrics() {
	s.done = make(chan struct{})
	go s.publishFPS(s.done)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for conn := range s.connections {
		_ = conn.Close()
	}
	s.connections = map[*websocket.Conn]string{}
	if s.done != nil {
		close(s.done)
		s.done = nil
	}
	s.mu.Unlock()

	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) ActiveDeviceID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeDeviceID
}

func (s *Server) Metrics() Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metrics
}

// State reports the truthful pipeline state for the UI.
func (s *Server) State() StreamState {
	s.mu.RLock()
	active := s.activeDeviceID
	s.mu.RUnlock()

	last := s.lastFrameNS.Load()
	agoMs := int64(-1)
	if last > 0 {
		agoMs = time.Since(time.Unix(0, last)).Milliseconds()
		if agoMs < 0 {
			agoMs = 0
		}
	}

	state := "idle"
	switch {
	case agoMs >= 0 && agoMs < 2000:
		state = "streaming"
	case s.streamStartNS.Load() > 0 && time.Since(time.Unix(0, s.streamStartNS.Load())) < 30*time.Second:
		state = "starting"
	case active != "":
		state = "connected"
	}
	return StreamState{State: state, ActiveDevice: active, LastFrameAgoMs: agoMs}
}

func (s *Server) RequestKeyframe() {
	s.mu.RLock()
	requestKeyframe := s.requestKeyframe
	defer s.mu.RUnlock()
	for conn, deviceID := range s.connections {
		if deviceID == s.activeDeviceID {
			_ = conn.WriteJSON(map[string]string{"type": "stream.request_keyframe"})
		}
	}
	if requestKeyframe != nil {
		requestKeyframe()
	}
}

func (s *Server) SetKeyframeRequester(request func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestKeyframe = request
}

func (s *Server) HandlePeerText(sendJSON func(any) error, payload []byte, currentDeviceID string) string {
	return s.handleText(sendJSON, payload, currentDeviceID)
}

func (s *Server) HandlePeerFrame(payload []byte) {
	s.handleFrame(payload)
}

func (s *Server) DisconnectDevice(deviceID string) {
	s.mu.Lock()
	if s.activeDeviceID == deviceID {
		s.activeDeviceID = ""
	}
	s.mu.Unlock()

	if s.Events.OnDeviceDisconnected != nil {
		s.Events.OnDeviceDisconnected(deviceID)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var connectedDeviceID string
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		switch messageType {
		case websocket.TextMessage:
			connectedDeviceID = s.handleText(func(v any) error {
				return conn.WriteJSON(v)
			}, payload, connectedDeviceID)
			if connectedDeviceID != "" {
				s.bindConnection(conn, connectedDeviceID)
			}
		case websocket.BinaryMessage:
			s.handleFrame(payload)
		}
	}

	if connectedDeviceID != "" {
		s.disconnect(conn, connectedDeviceID)
	}
}

func (s *Server) handleText(sendJSON func(any) error, payload []byte, currentDeviceID string) string {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		_ = sendJSON(map[string]string{"type": "error", "message": "invalid json"})
		return currentDeviceID
	}

	switch envelope.Type {
	case "pair.verify":
		var message PairVerifyMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			_ = sendJSON(map[string]string{"type": "error", "message": "invalid pair.verify"})
			return currentDeviceID
		}
		success, err := s.pairing.VerifyPairing(message.Token, pairing.DeviceHandshake{
			ID:             message.Device.ID,
			Name:           message.Device.Name,
			Model:          message.Device.Model,
			Manufacturer:   message.Device.Manufacturer,
			Platform:       message.Device.Platform,
			OSName:         message.Device.OSName,
			OSVersion:      message.Device.OSVersion,
			AndroidVersion: message.Device.AndroidVersion,
			StreamCapable:  streamCapable(message.Capabilities),
		})
		if err != nil {
			_ = sendJSON(map[string]string{"type": "pair.failed", "message": err.Error()})
			return currentDeviceID
		}
		_ = sendJSON(success)
		s.activateDevice(success.Device.ID)
		log.Printf("[HANDSHAKE] pair.verify device=%s platform=%s name=%q", success.Device.ID, success.Device.Platform, success.Device.Name)
		if s.Events.OnDeviceConnected != nil {
			s.Events.OnDeviceConnected(success.Device)
		}
		return success.Device.ID

	case "device.reconnect":
		var message ReconnectMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			_ = sendJSON(map[string]string{"type": "error", "message": "invalid reconnect"})
			return currentDeviceID
		}
		device, err := s.pairing.VerifyReconnect(message.DeviceID, message.TrustSecret)
		if err != nil {
			_ = sendJSON(map[string]string{"type": "reconnect.failed", "message": err.Error()})
			return currentDeviceID
		}
		_ = sendJSON(map[string]any{"type": "reconnect.success", "device": device})
		s.activateDevice(device.ID)
		log.Printf("[HANDSHAKE] device.reconnect device=%s name=%q", device.ID, device.Name)
		if s.Events.OnDeviceConnected != nil {
			s.Events.OnDeviceConnected(device)
		}
		return device.ID

	case "stream.start":
		var message StreamStartMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			return currentDeviceID
		}
		s.streamStartNS.Store(time.Now().UnixNano())
		s.mu.Lock()
		s.metrics.Codec = fallback(message.Codec, "H264")
		s.metrics.Resolution = resolution(message.Width, message.Height)
		s.metrics.UpdatedAt = time.Now().UTC()
		s.mu.Unlock()
		log.Printf("[STREAM_START] codec=%s %dx%d fps=%d", fallback(message.Codec, "H264"), message.Width, message.Height, message.TargetFPS)
		return currentDeviceID
	}

	return currentDeviceID
}

func (s *Server) bindConnection(conn *websocket.Conn, deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[conn] = deviceID
	s.activeDeviceID = deviceID
}

func (s *Server) activateDevice(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeDeviceID = deviceID
}

func (s *Server) disconnect(conn *websocket.Conn, deviceID string) {
	s.mu.Lock()
	delete(s.connections, conn)
	if s.activeDeviceID == deviceID {
		s.activeDeviceID = ""
	}
	s.mu.Unlock()

	if s.Events.OnDeviceDisconnected != nil {
		s.Events.OnDeviceDisconnected(deviceID)
	}
}

func (s *Server) handleFrame(payload []byte) {
	frame, err := ParseFramePacket(payload)
	if err != nil {
		log.Printf("[FRAME] rejected: %v (len=%d)", err, len(payload))
		return
	}

	receivedAt := time.Now()
	if s.lastFrameNS.Swap(receivedAt.UnixNano()) == 0 {
		log.Printf("[FRAME_RECEIVED] first frame key=%v bytes=%d ts=%d", frame.KeyFrame, len(frame.Payload), frame.TimestampNS)
	}
	frames := atomic.AddUint64(&s.frameCounter, 1)
	if n := atomic.AddUint64(&s.loggedFrames, 1); n%600 == 0 {
		lat := int64(0)
		if frame.TimestampNS > 0 {
			lat = receivedAt.Sub(time.Unix(0, frame.TimestampNS)).Milliseconds()
		}
		log.Printf("[FRAME] total=%d key=%v bytes=%d latency=%dms", frames, frame.KeyFrame, len(frame.Payload), lat)
	}
	latency := 0
	if frame.TimestampNS > 0 {
		latency = int(receivedAt.Sub(time.Unix(0, frame.TimestampNS)).Milliseconds())
		if latency < 0 {
			latency = 0
		}
	}

	s.mu.Lock()
	s.metrics.Frames = frames
	s.metrics.LatencyMS = latency
	s.metrics.UpdatedAt = receivedAt.UTC()
	s.mu.Unlock()

	if s.Events.OnFrame != nil {
		s.Events.OnFrame(FrameEvent{
			Codec:        "H264",
			Data:         base64.StdEncoding.EncodeToString(frame.Payload),
			KeyFrame:     frame.KeyFrame,
			TimestampNS:  frame.TimestampNS,
			ReceivedAtNS: receivedAt.UnixNano(),
		})
	}
}

func (s *Server) publishFPS(done <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var last uint64
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}

		current := atomic.LoadUint64(&s.frameCounter)
		fps := int(current - last)
		last = current

		s.mu.Lock()
		s.metrics.FPS = fps
		s.metrics.UpdatedAt = time.Now().UTC()
		metrics := s.metrics
		s.mu.Unlock()

		if s.Events.OnMetrics != nil {
			s.Events.OnMetrics(metrics)
		}
	}
}

func fallback(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func resolution(width, height int) string {
	if width <= 0 || height <= 0 {
		return "Auto"
	}
	return jsonNumber(width) + "x" + jsonNumber(height)
}

func jsonNumber(value int) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func streamCapable(capabilities map[string]any) bool {
	if value, ok := capabilities["canStream"].(bool); ok {
		return value
	}
	if encoder, ok := capabilities["encoder"].(string); ok && encoder == "none" {
		return false
	}
	return true
}
