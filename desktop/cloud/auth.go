package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const (
	callbackPortBase = 8971
	callbackPorts    = 8
	loginTimeout     = 3 * time.Minute
)

type tokenResult struct {
	params map[string]string
	err    error
}

// LoginWithDiscord opens the default browser, waits for Supabase to redirect
// back to a local loopback listener, and stores the resulting session.
// Supabase must allow `http://localhost:<port>/auth/callback` as a redirect URL.
func (c *Client) LoginWithDiscord(ctx context.Context) error {
	resultCh := make(chan tokenResult, 1)
	shutdown := make(chan struct{})

	mux := http.NewServeMux()
	var server *http.Server
	var port int

	page := func(status, message string) string {
		return "<!doctype html><html><head><meta charset='utf-8'><title>PeeperPhone</title>" +
			"<style>body{background:#0a0b0a;color:#c8f5a0;font-family:ui-monospace,monospace;display:grid;place-items:center;height:100vh;margin:0}p{letter-spacing:.2em;text-transform:uppercase;font-size:12px}</style></head>" +
			"<body><div style='text-align:center'><p>" + html.EscapeString(message) + "</p></div></body></html>"
	}

	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Tokens arrive in the fragment (#...), which never reaches the server.
		fmt.Fprint(w, `<script>
			const p = new URLSearchParams(location.hash.slice(1));
			if (!p.get('access_token')) { location.replace('/token?' + p.toString()); }
			else if (p.get('error')) { location.replace('/token?error=' + encodeURIComponent(p.get('error')) + '&error_description=' + encodeURIComponent(p.get('error_description') || '')); }
			else { location.replace('/token?' + p.toString()); }
			</script>`)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		q := r.URL.Query()
		if q.Get("error") != "" {
			fmt.Fprint(w, page("failed", "Login failed: "+q.Get("error_description")))
			resultCh <- tokenResult{err: errors.New(q.Get("error") + ": " + q.Get("error_description"))}
			return
		}
		access := q.Get("access_token")
		refresh := q.Get("refresh_token")
		expiresIn, _ := strconv.ParseInt(q.Get("expires_in"), 10, 64)
		if access == "" {
			fmt.Fprint(w, page("failed", "Login callback missing tokens"))
			resultCh <- tokenResult{err: errors.New("callback missing access_token")}
			return
		}
		user, err := c.fetchUser(access)
		if err != nil {
			fmt.Fprint(w, page("failed", "Could not load profile"))
			resultCh <- tokenResult{err: err}
			return
		}
		c.SetTokens(access, refresh, time.Now().Unix()+expiresIn-60, user)
		fmt.Fprint(w, page("ok", "Login complete. Return to PeeperPhone."))
		resultCh <- tokenResult{params: map[string]string{"ok": "1"}}
	})

	for i := 0; i < callbackPorts; i++ {
		port = callbackPortBase + i
		listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err != nil {
			continue
		}
		server = &http.Server{Handler: mux}
		go func() {
			_ = server.Serve(listener)
			close(shutdown)
		}()
		break
	}
	if server == nil {
		return fmt.Errorf("no free local port in range %d-%d", callbackPortBase, callbackPortBase+callbackPorts-1)
	}
	defer func() {
		_ = server.Close()
	}()

	redirectURI := fmt.Sprintf("http://localhost:%d/auth/callback", port)
	authURL := c.AuthorizeURL(redirectURI)
	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("could not open browser: %w (open manually: %s)", err, authURL)
	}

	timeout := time.After(loginTimeout)
	select {
	case res := <-resultCh:
		if res.err != nil {
			return res.err
		}
		return nil
	case <-timeout:
		return errors.New("login timed out, try again")
	case <-ctx.Done():
		return ctx.Err()
	case <-shutdown:
		return errors.New("local auth listener stopped unexpectedly")
	}
}

func (c *Client) fetchUser(accessToken string) (User, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := (&http.Client{Timeout: sessionTTL}).Do(req)
	if err != nil {
		return User{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("profile fetch failed (%s)", strings.TrimSpace(resp.Status))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return User{}, err
	}
	return user, nil
}
