package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// CamFrame is one JPEG frame broadcast by the phone over Supabase Realtime.
type CamFrame struct {
	J string `json:"j"`
	W int    `json:"w"`
	H int    `json:"h"`
}

// StreamCam subscribes to the pairing room's realtime channel and forwards
// camera frames until ctx is cancelled. Auto-reconnects with capped backoff.
// onError receives the first failure only.
func (c *Client) StreamCam(ctx context.Context, code string, onFrame func(CamFrame), onError func(error)) {
	go func() {
		attempt := 0
		for {
			if ctx.Err() != nil {
				return
			}
			err := c.camSession(ctx, code, onFrame)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				attempt++
				if attempt == 1 && onError != nil {
					onError(err)
				}
			}
			delay := time.Duration(attempt) * time.Second
			if delay > 10*time.Second {
				delay = 10 * time.Second
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}
	}()
}

func (c *Client) camSession(ctx context.Context, code string, onFrame func(CamFrame)) error {
	token, err := c.AccessToken()
	if err != nil {
		return err
	}

	wsURL := strings.Replace(c.baseURL, "https://", "wss://", 1) +
		"/realtime/v1/websocket?apikey=" + url.QueryEscape(c.apiKey) + "&vsn=1.0"

	dialer := websocket.Dialer{HandshakeTimeout: sessionTTL}
	conn, _, err := dialer.DialContext(ctx, wsURL, http.Header{})
	if err != nil {
		return err
	}
	defer conn.Close()

	var writeMu sync.Mutex
	send := func(v interface{}) error {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(websocket.TextMessage, raw)
	}

	topic := "realtime:room:" + code
	if err := send(map[string]interface{}{
		"topic": topic,
		"event": "phx_join",
		"ref":   "1",
		"payload": map[string]interface{}{
			"access_token": token,
			"config": map[string]interface{}{
				"broadcast": map[string]interface{}{"self": false},
				"presence":  map[string]interface{}{"key": ""},
			},
		},
	}); err != nil {
		return err
	}

	done := make(chan struct{})
	defer close(done)
	go func() {
		hb := time.NewTicker(20 * time.Second)
		defer hb.Stop()
		ref := 2
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				writeMu.Lock()
				_ = conn.Close()
				writeMu.Unlock()
				return
			case <-hb.C:
				_ = send(map[string]interface{}{
					"topic": "phoenix", "event": "heartbeat", "ref": ref, "payload": map[string]interface{}{},
				})
				ref++
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var msg struct {
			Event   string `json:"event"`
			Payload struct {
				Event   string   `json:"event"`
				Payload CamFrame `json:"payload"`
			} `json:"payload"`
		}
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		if msg.Event == "broadcast" && msg.Payload.Event == "cam" && msg.Payload.Payload.J != "" {
			onFrame(msg.Payload.Payload)
		}
	}
}
