package mainapp

import (
	"context"
	"errors"
	"fmt"

	"voixpe3per/desktop/network"
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
}

type Snapshot struct {
	Pairing pairing.SessionSnapshot `json:"pairing"`
	Devices []pairing.DeviceView    `json:"devices"`
	Metrics streaming.Metrics       `json:"metrics"`
}

func NewApp() *App {
	store := pairing.NewFileStore("")
	lan := network.NewLANDetector()
	pairingService := pairing.NewService(lan, store, 8080)
	server := streaming.NewServer(":8080", pairingService)

	app := &App{
		pairing: pairingService,
		server:  server,
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
	if err := a.pairing.StartSession(); err != nil {
		runtime.LogErrorf(ctx, "pairing session failed: %v", err)
	}
	if err := a.server.Start(); err != nil {
		runtime.LogErrorf(ctx, "stream server failed: %v", err)
	}
	a.startRelayIfConfigured(ctx)
	a.emitSnapshot()
}

func (a *App) OnShutdown(ctx context.Context) {
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

func (a *App) RefreshPairing() (pairing.SessionSnapshot, error) {
	if err := a.pairing.StartSession(); err != nil {
		return pairing.SessionSnapshot{}, err
	}
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

func (a *App) snapshot() Snapshot {
	return Snapshot{
		Pairing: a.pairing.Snapshot(),
		Devices: a.pairing.Devices(),
		Metrics: a.server.Metrics(),
	}
}

func (a *App) emitSnapshot() {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventSnapshot, a.snapshot())
	}
}

func (a *App) startRelayIfConfigured(ctx context.Context) {
	snapshot := a.pairing.Snapshot()
	if snapshot.Mode != pairing.ModeRelay || snapshot.RelayURL == "" || snapshot.Room == "" {
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
