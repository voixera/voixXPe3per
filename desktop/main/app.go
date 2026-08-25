package mainapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"voixpe3per/desktop/cloud"
	"voixpe3per/desktop/pairing"
	"voixpe3per/desktop/streaming"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventSnapshot = "app.snapshot"
	eventFrame    = "stream.frame"
	eventMetrics  = "stream.metrics"
)

type App struct {
	ctx     context.Context
	pairing *pairing.Service
	server  *streaming.Server
	relay   *streaming.RelayClient
	cloud   *cloud.Client

	watchMu    sync.Mutex
	cancelWatch context.CancelFunc

	camMu      sync.Mutex
	camCancel  context.CancelFunc
	camActive  bool
}

type Snapshot struct {
	Pairing   pairing.SessionSnapshot `json:"pairing"`
	Devices   []pairing.DeviceView    `json:"devices"`
	Metrics   streaming.Metrics       `json:"metrics"`
	Auth      cloud.Identity          `json:"auth"`
	CamActive bool                    `json:"camActive"`
}

func NewApp() *App {
	store := pairing.NewFileStore("")
	pairingService := pairing.NewService(store, 8080)
	server := streaming.NewServer(":8080", pairingService)

	app := &App{
		pairing: pairingService,
		server:  server,
	}

	if cloud.Configured() {
		app.cloud = cloud.NewClient()
		app.cloud.Restore()
	}

	server.Events = streaming.Events{
		OnDeviceConnected: func(device pairing.DeviceView) {
			app.emitSnapshot()
		},
		OnDeviceDisconnected: func(deviceID string) {
			app.pairing.MarkDeviceOffline(deviceID)
			app.emitSnapshot()
		},
		OnFrame: func(frame streaming.FrameEvent) {
			if app.ctx != nil {
				runtime.EventsEmit(app.ctx, eventFrame, frame)
			}
		},
		OnMetrics: func(metrics streaming.Metrics) {
			if app.ctx != nil {
				runtime.EventsEmit(app.ctx, eventMetrics, metrics)
			}
		},
	}

	return app
}

func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
	a.refreshPairingInternal()
	if a.pairing.Snapshot().Mode == pairing.ModeRelay {
		a.server.StartRelay()
	} else if err := a.server.Start(); err != nil {
		runtime.LogErrorf(ctx, "stream server failed: %v", err)
	}
	a.startRelayIfConfigured(ctx)
	a.emitSnapshot()
}

func (a *App) OnShutdown(ctx context.Context) {
	a.stopWatch()
	if a.relay != nil {
		a.relay.Shutdown()
	}
	if err := a.server.Shutdown(ctx); err != nil {
		runtime.LogErrorf(ctx, "stream server shutdown failed: %v", err)
	}
}

func (a *App) GetSnapshot() (Snapshot, error) {
	return a.snapshot(), nil
}

// LoginWithDiscord opens the browser for the Supabase Discord OAuth flow and
// blocks until the loopback callback receives tokens.
func (a *App) LoginWithDiscord() error {
	if a.cloud == nil {
		return errors.New("supabase is not configured (set PEEPER_SUPABASE_URL and PEEPER_SUPABASE_ANON_KEY)")
	}
	if a.ctx == nil {
		return errors.New("app not ready")
	}
	if err := a.cloud.LoginWithDiscord(a.ctx); err != nil {
		runtime.LogErrorf(a.ctx, "discord login failed: %v", err)
		return err
	}
	runtime.LogInfo(a.ctx, "discord login complete")
	a.refreshPairingInternal()
	a.emitSnapshot()
	return nil
}

func (a *App) Logout() error {
	if a.cloud != nil {
		a.cloud.Logout()
	}
	a.stopWatch()
	a.refreshPairingInternal()
	a.startRelayIfConfigured(a.ctx)
	a.emitSnapshot()
	return nil
}

func (a *App) RefreshPairing() (pairing.SessionSnapshot, error) {
	a.refreshPairingInternal()
	a.startRelayIfConfigured(a.ctx)
	a.emitSnapshot()
	return a.pairing.Snapshot(), nil
}

func (a *App) ForgetDevice(deviceID string) error {
	if deviceID == "" {
		return errors.New("device id is required")
	}
	if err := a.pairing.ForgetDevice(deviceID); err != nil {
		return err
	}
	a.emitSnapshot()
	return nil
}

func (a *App) RefreshStream() error {
	if a.server.ActiveDeviceID() == "" {
		return fmt.Errorf("no active device connected")
	}
	a.server.RequestKeyframe()
	return nil
}

func (a *App) ToggleFullscreen() {
	if a.ctx != nil {
		if runtime.WindowIsFullscreen(a.ctx) {
			runtime.WindowUnfullscreen(a.ctx)
			return
		}
		runtime.WindowFullscreen(a.ctx)
	}
}

// refreshPairingInternal starts either a cloud session or the legacy local/relay one.
func (a *App) refreshPairingInternal() {
	a.stopWatch()

	if a.cloudReadyAndLoggedIn() {
		code, err := a.pairing.StartCloudSession(a.hostLabel())
		if err != nil {
			runtime.LogErrorf(a.ctx, "cloud pairing session failed: %v", err)
			return
		}
		if err := a.cloud.CreatePairingSession(context.Background(), code, a.hostLabel()); err != nil {
			runtime.LogErrorf(a.ctx, "supabase session insert failed: %v", err)
		}
		a.watchCloudApprovals()
		return
	}

	if err := a.pairing.StartSession(); err != nil && a.ctx != nil {
		runtime.LogErrorf(a.ctx, "pairing session failed: %v", err)
	}
}

func (a *App) hostLabel() string {
	if a.cloud != nil {
		if name := a.cloud.Identity().Name; name != "" {
			return name + " Desktop"
		}
	}
	return "PeeperPhone Desktop"
}

func (a *App) cloudReadyAndLoggedIn() bool {
	return a.cloud != nil && cloud.Configured() && a.cloud.LoggedIn()
}

// watchCloudApprovals polls the Supabase row until a device approves.
func (a *App) watchCloudApprovals() {
	a.stopWatch()
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelWatch = cancel

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			snapshot := a.pairing.Snapshot()
			if snapshot.Mode != pairing.ModeCloud || snapshot.Room == "" {
				return
			}
			session, err := a.cloud.GetPairingSession(ctx, snapshot.Room)
			if err != nil || session.Status != "approved" || len(session.Device) == 0 {
				continue
			}

			var handshake pairing.DeviceHandshake
			if err := json.Unmarshal(session.Device, &handshake); err != nil {
				continue
			}
			if _, err := a.pairing.VerifyPairing("", handshake); err != nil {
				runtime.LogErrorf(a.ctx, "approved device rejected: %v", err)
				continue
			}
			_ = a.cloud.ConsumePairingSession(context.Background(), snapshot.Room)
			runtime.LogInfo(a.ctx, "device approved via discord account")
			a.startCamFeed(snapshot.Room)
			a.stopWatch()
			a.emitSnapshot()
			return
		}
	}()
}

func (a *App) stopWatch() {
	a.watchMu.Lock()
	defer a.watchMu.Unlock()
	if a.cancelWatch != nil {
		a.cancelWatch()
		a.cancelWatch = nil
	}
}

// startCamFeed listens for the phone's camera broadcast on the room channel.
func (a *App) startCamFeed(code string) {
	if a.cloud == nil || code == "" {
		return
	}
	a.stopCam()
	ctx, cancel := context.WithCancel(context.Background())
	a.camMu.Lock()
	a.camCancel = cancel
	a.camActive = true
	a.camMu.Unlock()

	frames := 0
	loggedErr := false
	a.cloud.StreamCam(ctx, code, func(frame cloud.CamFrame) {
		frames++
		if frames == 1 {
			runtime.LogInfo(a.ctx, "camera feed connected, receiving frames")
			a.emitSnapshot()
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "cam.frame", frame)
		}
	}, func(err error) {
		if !loggedErr && a.ctx != nil {
			loggedErr = true
			runtime.LogErrorf(a.ctx, "camera realtime stream failed: %v", err)
		}
	})
}

func (a *App) stopCam() {
	a.camMu.Lock()
	defer a.camMu.Unlock()
	if a.camCancel != nil {
		a.camCancel()
		a.camCancel = nil
	}
	a.camActive = false
}

func (a *App) snapshot() Snapshot {
	auth := cloud.Identity{CloudReady: cloud.Configured()}
	if a.cloud != nil {
		auth = a.cloud.Identity()
		auth.CloudReady = true
	}
	return Snapshot{
		Pairing:   a.pairing.Snapshot(),
		Devices:   a.pairing.Devices(),
		Metrics:   a.server.Metrics(),
		Auth:      auth,
		CamActive: a.camActive,
	}
}

func (a *App) emitSnapshot() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventSnapshot, a.snapshot())
	}
}

func (a *App) startRelayIfConfigured(ctx context.Context) {
	snapshot := a.pairing.Snapshot()
	if snapshot.Mode == pairing.ModeCloud {
		if a.relay != nil {
			a.relay.Shutdown()
			a.relay = nil
		}
		return
	}
	if snapshot.Mode != pairing.ModeRelay || snapshot.RelayURL == "" || snapshot.Room == "" {
		if a.relay != nil {
			a.relay.Shutdown()
			a.relay = nil
		}
		return
	}

	if a.relay != nil {
		a.relay.Shutdown()
	}

	a.relay = streaming.NewRelayClient(snapshot.RelayURL, snapshot.Room, a.server)
	if err := a.relay.Start(); err != nil {
		runtime.LogErrorf(ctx, "relay connection failed: %v", err)
	}
}
