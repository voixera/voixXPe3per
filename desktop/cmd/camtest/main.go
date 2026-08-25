// camtest verifies Supabase Realtime broadcast delivery between two raw
// websocket clients, mirroring phone(sender)/desktop(receiver) exactly.
// Usage: go run ./cmd/camtest -url <supabase url> -key <anon key> [-room X]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type msg struct {
	Event   string          `json:"event"`
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
	Status  string          `json:"status,omitempty"`
}

type inner struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func main() {
	base := flag.String("url", "", "supabase base url")
	key := flag.String("key", "", "anon key")
	room := flag.String("room", "CAMTEST1", "room code")
	flag.Parse()

	topic := "realtime:room:" + strings.ToUpper(*room)
	wsBase := strings.Replace(strings.TrimRight(*base, "/"), "https://", "wss://", 1)
	endpoint := wsBase + "/realtime/v1/websocket?apikey=" + url.QueryEscape(*key) + "&vsn=1.0"

	dial := func(name string) *websocket.Conn {
		variants := []struct {
			label string
			dialer websocket.Dialer
			header http.Header
		}{
			{"plain", websocket.Dialer{HandshakeTimeout: 10 * time.Second}, nil},
			{"proto-phoenix", websocket.Dialer{HandshakeTimeout: 10 * time.Second, Subprotocols: []string{"phoenix"}}, nil},
			{"proto-both", websocket.Dialer{HandshakeTimeout: 10 * time.Second, Subprotocols: []string{"phoenix", "v1.api.supabase.co"}}, nil},
			{"hdr-apikey", websocket.Dialer{HandshakeTimeout: 10 * time.Second}, http.Header{"apikey": []string{*key}}},
		}
		for _, v := range variants {
			c, resp, err := v.dialer.Dial(endpoint, v.header)
			if err == nil {
				fmt.Printf("[%s] dialed ok via %s\n", name, v.label)
				return c
			}
			status := ""
			if resp != nil {
				status = resp.Status
			}
			fmt.Printf("[%s] variant %s failed: %v %s\n", name, v.label, err, status)
		}
		return nil
	}

	join := []string{"A-sender", "B-receiver"}
	conns := make([]*websocket.Conn, 2)
	for i, name := range join {
		c := dial(name)
		if c == nil {
			return
		}
		conns[i] = c
		sendJoin(c, name, topic, *key)
	}

	var got atomic.Int32
	done := make(chan struct{})

	// Reader for B (receiver)
	go func() {
		for {
			_, raw, err := conns[1].ReadMessage()
			if err != nil {
				fmt.Printf("[B-receiver] read closed: %v\n", err)
				close(done)
				return
			}
			var m msg
			json.Unmarshal(raw, &m)
			switch m.Event {
			case "phx_reply":
				fmt.Printf("[B-receiver] reply status=%s topic=%s\n", m.Status, m.Topic)
			case "phx_error":
				fmt.Printf("[B-receiver] JOIN ERROR topic=%s payload=%s\n", m.Topic, truncate(raw))
				close(done)
				return
			default:
				fmt.Printf("[B-receiver] event=%s topic=%s\n", m.Event, m.Topic)
			}
			if m.Event == "broadcast" {
				var p inner
				json.Unmarshal(m.Payload, &p)
				if p.Event == "cam" {
					n := got.Add(1)
					fmt.Printf("[B-receiver] GOT CAM FRAME #%d (%d bytes)\n", n, len(p.Payload))
					if n >= 3 {
						close(done)
						return
					}
				}
			}
		}
	}()

	// Reader for A (just logs replies)
	go func() {
		for {
			_, raw, err := conns[0].ReadMessage()
			if err != nil {
				fmt.Printf("[A-sender] read closed: %v\n", err)
				return
			}
			var m msg
			json.Unmarshal(raw, &m)
			if m.Event == "phx_reply" && strings.Contains(string(raw), "phx_join") {
				fmt.Printf("[A-sender] reply status=%s\n", m.Status)
			}
			if m.Event == "phx_error" {
				fmt.Printf("[A-sender] JOIN ERROR payload=%s\n", truncate(raw))
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)

	payload := map[string]interface{}{
		"type":    "broadcast",
		"event":   "cam",
		"payload": map[string]interface{}{"j": "dGVzdA==", "w": 8, "h": 8},
	}
	for i := 0; i < 12; i++ {
		raw, _ := json.Marshal(payload)
		if err := conns[0].WriteMessage(websocket.TextMessage, raw); err != nil {
			fmt.Printf("[A-sender] send FAILED: %v\n", err)
			break
		}
		fmt.Printf("[A-sender] sent broadcast #%d\n", i+1)
		time.Sleep(400 * time.Millisecond)
	}

	select {
	case <-done:
	case <-time.After(6 * time.Second):
	}
	fmt.Printf("\nRESULT: receiver got %d/3+ frames -> %s\n", got.Load(), passFail(got.Load()))
}

func passFail(n int32) string {
	if n >= 3 {
		return "TRANSPORT WORKS"
	}
	return "TRANSPORT BROKEN"
}

func sendJoin(c *websocket.Conn, name, topic, accessToken string) {
	m := map[string]interface{}{
		"topic": topic,
		"event": "phx_join",
		"ref":   "1",
		"payload": map[string]interface{}{
			"access_token": accessToken,
			"config": map[string]interface{}{
				"broadcast": map[string]interface{}{"self": false},
				"presence":  map[string]interface{}{"key": ""},
			},
		},
	}
	raw, _ := json.Marshal(m)
	if err := c.WriteMessage(websocket.TextMessage, raw); err != nil {
		fmt.Printf("[%s] join write FAILED: %v\n", name, err)
	} else {
		fmt.Printf("[%s] joined %s\n", name, topic)
	}
}

func truncate(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
