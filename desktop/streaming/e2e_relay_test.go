package streaming

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"voixpe3per/desktop/pairing"

	"github.com/gorilla/websocket"
)

// fakeRelay mimics relay/src/server.js fanout: relay.join registers the peer,
// everything else is rebroadcast verbatim to room peers (binary kept binary).
type fakeRelay struct {
	mu    sync.Mutex
	rooms map[string]map[*websocket.Conn]struct{}
}

func (f *fakeRelay) handler(t *testing.T, w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		mt, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var join struct {
			Type string `json:"type"`
			Room string `json:"room"`
		}
		isJoin := mt == websocket.TextMessage && json.Unmarshal(payload, &join) == nil && join.Type == "relay.join"
		if isJoin {
			f.mu.Lock()
			if f.rooms == nil {
				f.rooms = map[string]map[*websocket.Conn]struct{}{}
			}
			if f.rooms[join.Room] == nil {
				f.rooms[join.Room] = map[*websocket.Conn]struct{}{}
			}
			f.rooms[join.Room][conn] = struct{}{}
			f.mu.Unlock()
			continue
		}
		room := f.roomOf(conn)
		if room == "" {
			continue
		}
		f.mu.Lock()
		for peer := range f.rooms[room] {
			if peer != conn {
				_ = peer.WriteMessage(mt, payload)
			}
		}
		f.mu.Unlock()
	}
}

func (f *fakeRelay) roomOf(target *websocket.Conn) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for room, peers := range f.rooms {
		if _, ok := peers[target]; ok {
			return room
		}
	}
	return ""
}

// TestRelayPipelineEndToEnd drives CONNECT → JOIN → HANDSHAKE → STREAM_START
// → FRAME_RECEIVED exactly like the Android APK does through the public relay.
func TestRelayPipelineEndToEnd(t *testing.T) {
	// Fake relay listener.
	relay := &fakeRelay{}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		relay.handler(t, w, r)
	})}
	go func() { _ = srv.Serve(listener) }()
	relayURL := "ws://" + listener.Addr().String() + "/ws"

	// Desktop pairing session bound to the fake relay.
	t.Setenv("VOIXPE3PER_RELAY_URL", relayURL)
	store := pairing.NewFileStore(filepath.Join(t.TempDir(), "devices.json"))
	svc := pairing.NewService(store, 8080)
	if err := svc.StartSession(); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	room := svc.Snapshot().Room

	streamServer := NewServer("127.0.0.1:0", svc)
	frames := make(chan FrameEvent, 16)
	streamServer.Events = Events{
		OnDeviceConnected: func(pairing.DeviceView) {},
		OnFrame:           func(f FrameEvent) { frames <- f },
	}
	streamServer.StartRelay()
	defer func() { _ = streamServer.Shutdown(context.Background()) }()

	rc := NewRelayClient(relayURL, room, streamServer)
	if err := rc.Start(); err != nil {
		t.Fatalf("relay client start: %v", err)
	}
	defer rc.Shutdown()

	// Phone side.
	phone, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer phone.Close()
	joinRoom(t, phone, room)

	waitFor(t, "desktop joined room", 3*time.Second, func() bool {
		return len(relay.peers(room)) == 2
	})

	// HANDSHAKE: pair.verify with the session token (relay mode requires it).
	payload, _ := json.Marshal(map[string]any{
		"type":  "pair.verify",
		"token": svc.Snapshot().Token,
		"device": map[string]any{
			"id": "test-device-1", "name": "Test Phone", "model": "Pixel",
			"platform": "android", "osName": "Android", "osVersion": "14",
		},
		"capabilities": map[string]any{"canStream": true},
	})
	if err := phone.WriteMessage(websocket.TextMessage, payload); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "handshake activates device", 3*time.Second, func() bool {
		return streamServer.ActiveDeviceID() == "test-device-1" && streamServer.State().State == "connected"
	})

	// STREAM_START.
	startPayload, _ := json.Marshal(map[string]any{
		"type": "stream.start", "codec": "H264", "width": 1080, "height": 2400, "targetFps": 60,
	})
	if err := phone.WriteMessage(websocket.TextMessage, startPayload); err != nil {
		t.Fatal(err)
	}

	// FRAME_RECEIVED: VX packet.
	packet := buildFramePacket(true, 123, []byte("h264payload"))
	if err := phone.WriteMessage(websocket.BinaryMessage, packet); err != nil {
		t.Fatal(err)
	}

	select {
	case <-frames:
	case <-time.After(3 * time.Second):
		t.Fatal("frame never reached the desktop server")
	}
	if got := streamServer.Metrics().Resolution; got != "1080x2400" {
		t.Fatalf("resolution = %q, want 1080x2400", got)
	}
	waitFor(t, "state becomes streaming", 2*time.Second, func() bool {
		return streamServer.State().State == "streaming"
	})
}

func joinRoom(t *testing.T, conn *websocket.Conn, room string) {
	t.Helper()
	msg, _ := json.Marshal(map[string]any{"type": "relay.join", "role": "android", "room": room})
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatal(err)
	}
}

func (f *fakeRelay) peers(room string) []*websocket.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*websocket.Conn, 0, len(f.rooms[room]))
	for peer := range f.rooms[room] {
		out = append(out, peer)
	}
	return out
}

func waitFor(t *testing.T, what string, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", what)
}

func buildFramePacket(key bool, ts int64, payload []byte) []byte {
	packet := make([]byte, 12+len(payload))
	packet[0] = 'V'
	packet[1] = 'X'
	packet[2] = 1
	if key {
		packet[3] = 1
	}
	binary.BigEndian.PutUint64(packet[4:12], uint64(ts))
	copy(packet[12:], payload)
	return packet
}
