package streaming

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RelayClient keeps a persistent connection to the public WSS relay and
// rejoins the pairing room after every disconnect (Render cold starts,
// network blips). Without the retry loop a single failed dial left the
// desktop deaf forever: phone frames went into an empty room.
type RelayClient struct {
	url    string
	room   string
	server *Server

	mu       sync.Mutex
	conn     *websocket.Conn
	deviceID string
	done     chan struct{}
	joined   bool // at least one successful join since Start
}

func NewRelayClient(url, room string, server *Server) *RelayClient {
	return &RelayClient{
		url:    url,
		room:   room,
		server: server,
		done:   make(chan struct{}),
	}
}

func (c *RelayClient) Start() error {
	if c.url == "" || c.room == "" {
		return fmt.Errorf("relay url and room are required")
	}

	go c.runLoop()
	return nil
}

// runLoop dials with backoff until Shutdown; every established connection
// joins the room and spawns a read loop.
func (c *RelayClient) runLoop() {
	backoff := time.Second
	for {
		select {
		case <-c.done:
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
		if err != nil {
			log.Printf("[RELAY] CONNECT failed (%v) — retrying in %s", err, backoff)
			select {
			case <-c.done:
				return
			case <-time.After(backoff):
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second

		log.Printf("[RELAY] CONNECT %s", c.url)
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		if err := c.writeJSON(map[string]any{
			"type": "relay.join",
			"role": "desktop",
			"room": c.room,
		}); err != nil {
			_ = conn.Close()
			continue
		}
		c.joined = true
		log.Printf("[RELAY] JOIN room=%s role=desktop", c.room)

		c.server.SetKeyframeRequester(func() {
			_ = c.writeJSON(map[string]string{"type": "stream.request_keyframe"})
		})

		c.readLoop(conn)
		_ = conn.Close()
		log.Printf("[RELAY] DISCONNECT — rejoining")
	}
}

func (c *RelayClient) Shutdown() {
	close(c.done)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	if c.deviceID != "" {
		c.server.DisconnectDevice(c.deviceID)
	}
}

func (c *RelayClient) readLoop(conn *websocket.Conn) {
	defer func() {
		c.mu.Lock()
		deviceID := c.deviceID
		c.deviceID = ""
		c.mu.Unlock()
		if deviceID != "" {
			c.server.DisconnectDevice(deviceID)
		}
	}()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	for {
		select {
		case <-c.done:
			return
		default:
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.TextMessage:
			if c.handleRelayControl(payload) {
				continue
			}
			c.mu.Lock()
			current := c.deviceID
			c.mu.Unlock()
			newID := c.server.HandlePeerText(c.writeJSON, payload, current)
			c.mu.Lock()
			c.deviceID = newID
			c.mu.Unlock()
		case websocket.BinaryMessage:
			c.server.HandlePeerFrame(payload)
		}
	}
}

func (c *RelayClient) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("relay is not connected")
	}
	return c.conn.WriteJSON(v)
}

func (c *RelayClient) handleRelayControl(payload []byte) bool {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return false
	}
	if !strings.HasPrefix(envelope.Type, "relay.") {
		return false
	}
	switch envelope.Type {
	case "relay.peer_left":
		c.mu.Lock()
		deviceID := c.deviceID
		c.deviceID = ""
		c.mu.Unlock()
		if deviceID != "" {
			c.server.DisconnectDevice(deviceID)
		}
	case "relay.error":
		log.Printf("[RELAY] error: %s", string(payload))
	}
	return true
}
