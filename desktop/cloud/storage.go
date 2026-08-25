package cloud

import (
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"time"
)

// PollCamJPEG fetches the phone's latest camera frame from the public
// peeper-cam Storage bucket and calls cb whenever the bytes change. This is
// plain HTTP (works where Realtime websockets are blocked by Cloudflare).
// ~3 frames/second effective.
func (c *Client) PollCamJPEG(ctx context.Context, room string, cb func(jpeg []byte)) {
	url := c.baseURL + "/storage/v1/object/public/peeper-cam/" + room + ".jpg"
	client := &http.Client{Timeout: 4 * time.Second}
	var lastHash [32]byte

	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url+"?cb="+time.Now().Format("150405.000000000"), nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil || len(raw) == 0 {
			continue
		}

		sum := sha256.Sum256(raw)
		if sum == lastHash {
			continue
		}
		lastHash = sum
		cb(raw)
	}
}
