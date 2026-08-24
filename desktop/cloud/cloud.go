package cloud

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	envURL     = "PEEPER_SUPABASE_URL"
	envURLAlt  = "SUPABASE_URL"
	envKey     = "PEEPER_SUPABASE_ANON_KEY"
	envKeyAlt  = "SUPABASE_ANON_KEY"
	sessionTTL = 5 * time.Second
)

var dotenvOnce sync.Once

// loadEnv reads the nearest .env file so launching the exe directly works.
// Real environment variables always win over .env values.
func loadEnv() {
	dotenvOnce.Do(func() {
		for _, dir := range candidateDirs() {
			if applyDotEnv(filepath.Join(dir, ".env")) {
				return
			}
		}
	})
}

func candidateDirs() []string {
	var dirs []string
	if wd, err := os.Getwd(); err == nil {
		for i := 0; i < 4; i++ {
			dirs = append(dirs, wd)
			parent := filepath.Dir(wd)
			if parent == wd {
				break
			}
			wd = parent
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 3; i++ {
			dirs = append(dirs, dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return dirs
}

func applyDotEnv(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.Trim(strings.TrimSpace(key), `"'`)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return true
}

type Identity struct {
	LoggedIn    bool   `json:"loggedIn"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	ProviderID  string `json:"providerId"`
	CloudReady  bool   `json:"cloudReady"`
}

// RemoteSession mirrors one row of public.pairing_sessions.
type RemoteSession struct {
	Code         string          `json:"code"`
	Status       string          `json:"status"`
	Device       json.RawMessage `json:"device"`
	ApprovedAt   *time.Time      `json:"approved_at"`
}

type storedTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	User         User   `json:"user"`
}

type User struct {
	ID    string                 `json:"id"`
	Email string                 `json:"email"`
	Meta  map[string]interface{} `json:"user_metadata"`
}

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client

	mu       sync.Mutex
	tokens   storedTokens
	identity Identity
	file     string
}

func Configured() bool {
	loadEnv()
	return baseURLFromEnv() != "" && keyFromEnv() != ""
}

func NewClient() *Client {
	loadEnv()
	return &Client{
		baseURL: strings.TrimRight(baseURLFromEnv(), "/"),
		apiKey:  keyFromEnv(),
		http:    &http.Client{Timeout: sessionTTL},
	}
}

func baseURLFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(envURL)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(envURLAlt))
}

func keyFromEnv() string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv(envKeyAlt))
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "peeperphone")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

// Restore loads a saved Discord login from disk.
func (c *Client) Restore() {
	c.mu.Lock()
	defer c.mu.Unlock()
	dir, err := configDir()
	if err != nil {
		return
	}
	c.file = filepath.Join(dir, "session.json")
	raw, err := os.ReadFile(c.file)
	if err != nil {
		return
	}
	_ = json.Unmarshal(raw, &c.tokens)
	c.hydrateUserLocked()
}

func (c *Client) LoggedIn() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tokens.AccessToken != ""
}

func (c *Client) AccessToken() (string, error) {
	c.mu.Lock()
	tok := c.tokens.AccessToken
	exp := c.tokens.ExpiresAt
	c.mu.Unlock()

	if tok == "" {
		return "", errors.New("not logged in")
	}
	if exp > 0 && time.Now().Unix() < exp-30 {
		return tok, nil
	}
	if err := c.refresh(); err != nil {
		return "", err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tokens.AccessToken, nil
}

func (c *Client) refresh() error {
	c.mu.Lock()
	refreshToken := c.tokens.RefreshToken
	c.mu.Unlock()
	if refreshToken == "" {
		return errors.New("session expired, login again")
	}

	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	req, err := http.NewRequest("POST", c.baseURL+"/auth/v1/token?grant_type=refresh_token", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		c.mu.Lock()
		c.tokens = storedTokens{}
		c.saveLocked()
		c.mu.Unlock()
		return fmt.Errorf("refresh failed (%d)", resp.StatusCode)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		User         User   `json:"user"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens.AccessToken = payload.AccessToken
	c.tokens.RefreshToken = payload.RefreshToken
	if payload.ExpiresIn > 0 {
		c.tokens.ExpiresAt = time.Now().Unix() + payload.ExpiresIn - 60
	}
	if payload.User.ID != "" {
		c.tokens.User = payload.User
	}
	c.hydrateUserLocked()
	c.saveLocked()
	return nil
}

// hydrateUserLocked fills display fields from the JWT user metadata.
func (c *Client) hydrateUserLocked() {
	meta := c.tokens.User.Meta
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := meta[k]; ok {
				switch t := v.(type) {
				case string:
					if t != "" {
						return t
					}
				}
			}
		}
		return ""
	}
	name := get("full_name", "name", "user_name", "preferred_username")
	if name == "" {
		name = c.tokens.User.Email
	}
	c.identity = Identity{
		LoggedIn:   c.tokens.AccessToken != "",
		Email:      c.tokens.User.Email,
		Name:       name,
		Avatar:     get("avatar_url"),
		ProviderID: get("provider_id"),
		CloudReady: c.baseURL != "" && c.apiKey != "",
	}
}

func (c *Client) saveLocked() {
	if c.file == "" {
		dir, err := configDir()
		if err != nil {
			return
		}
		c.file = filepath.Join(dir, "session.json")
	}
	raw, err := json.Marshal(c.tokens)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.file, raw, 0o600)
}

func (c *Client) SetTokens(access, refresh string, expiresAt int64, user User) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = storedTokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    expiresAt,
		User:         user,
	}
	c.hydrateUserLocked()
	c.saveLocked()
}

func (c *Client) Identity() Identity {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.identity
}

func (c *Client) Logout() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = storedTokens{}
	c.saveLocked()
}

// --- REST helpers -------------------------------------------------------

func (c *Client) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	token, err := c.AccessToken()
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Prefer", "return=representation")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		return fmt.Errorf("supabase %s %s failed (%d): %s", method, path, resp.StatusCode, msg)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}

// AuthorizeURL builds the Supabase OAuth entry point for the Discord provider.
// Implicit flow: Supabase redirects back with tokens in the URL fragment.
func (c *Client) AuthorizeURL(redirectURI string) string {
	return c.baseURL + "/auth/v1/authorize?provider=discord&redirect_to=" + queryEscape(redirectURI)
}

func queryEscape(v string) string {
	return url.QueryEscape(v)
}

// CreatePairingSession inserts a waiting row into public.pairing_sessions.
func (c *Client) CreatePairingSession(ctx context.Context, code, hostName string) error {
	var created []map[string]interface{}
	return c.do(ctx, "POST",
		"/rest/v1/pairing_sessions",
		map[string]interface{}{
			"code":         code,
			"host_user_id": c.userID(),
			"host_name":    hostName,
			"status":       "waiting",
		},
		&created,
	)
}

func (c *Client) userID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tokens.User.ID
}

// GetPairingSession fetches the current remote session state.
func (c *Client) GetPairingSession(ctx context.Context, code string) (*RemoteSession, error) {
	var rows []RemoteSession
	err := c.do(ctx, "GET",
		"/rest/v1/pairing_sessions?select=code,status,device,approved_at&code=eq."+queryEscape(code),
		nil,
		&rows,
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("pairing session vanished")
	}
	return &rows[0], nil
}

// ConsumePairingSession marks the row as used so it cannot be replayed.
func (c *Client) ConsumePairingSession(ctx context.Context, code string) error {
	return c.do(ctx, "PATCH",
		"/rest/v1/pairing_sessions?code=eq."+queryEscape(code),
		map[string]interface{}{"status": "consumed"},
		nil,
	)
}
