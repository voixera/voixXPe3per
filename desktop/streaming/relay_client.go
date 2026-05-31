package streaming

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type RelayClient struct {
	url    string
	room   string
	server *Server

	mu       sync.Mutex
	conn     *websocket.Conn
	deviceID string
	done     chan struct{}
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

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	if err := c.writeJSON(map[string]any{
		"type": "relay.join",
		"role": "desktop",
		"room": c.room,
	}); err != nil {
		_ = conn.Close()
		return err
	}

	c.server.SetKeyframeRequester(func() {
		_ = c.writeJSON(map[string]string{"type": "stream.request_keyframe"})
	})

	go c.readLoop(conn)
	return nil
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
		if c.deviceID != "" {
			c.server.DisconnectDevice(c.deviceID)
		}
	}()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		switch messageType {
		case websocket.TextMessage:
			if isRelayControl(payload) {
				continue
			}
			c.deviceID = c.server.HandlePeerText(c.writeJSON, payload, c.deviceID)
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

func isRelayControl(payload []byte) bool {
	var envelope Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return false
	}
	return strings.HasPrefix(envelope.Type, "relay.")
}
