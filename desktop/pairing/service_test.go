package pairing

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(NewFileStore(filepath.Join(t.TempDir(), "devices.json")), 8080)
}

func handshake(id string) DeviceHandshake {
	return DeviceHandshake{ID: id, Name: "Dev " + id, Platform: "web", StreamCapable: false}
}

func TestDeviceLimit(t *testing.T) {
	s := newTestService(t)
	if _, err := s.StartCloudSession("host"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxDevices; i++ {
		if _, err := s.VerifyPairing("", handshake(string(rune('a'+i)))); err != nil {
			t.Fatalf("device %d rejected: %v", i, err)
		}
	}
	if _, err := s.VerifyPairing("", handshake("overflow")); !errors.Is(err, ErrDeviceLimit) {
		t.Fatalf("expected ErrDeviceLimit, got %v", err)
	}
	// Re-pairing a known device stays allowed at the cap.
	if _, err := s.VerifyPairing("", handshake("a")); err != nil {
		t.Fatalf("known device re-pair rejected: %v", err)
	}
}

func TestRestoreByRoomAndConnect(t *testing.T) {
	s := newTestService(t)
	code, err := s.StartCloudSession("host")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyPairing("", handshake("phone1")); err != nil {
		t.Fatal(err)
	}

	if err := s.startCloudSessionWithCode(code); err != nil {
		t.Fatal(err)
	}
	if snap := s.Snapshot(); snap.Room != code || snap.Mode != ModeCloud {
		t.Fatalf("restore mismatch: %+v", snap)
	}
	if !s.ConnectDevice("phone1") {
		t.Fatal("ConnectDevice failed for known device")
	}
	if s.ConnectDevice("ghost") {
		t.Fatal("ConnectDevice should fail for unknown device")
	}
	if !s.HasDevice("phone1") || s.HasDevice("ghost") {
		t.Fatal("HasDevice inconsistent")
	}
	devs := s.Devices()
	if len(devs) != 1 || devs[0].Status != DeviceConnected {
		t.Fatalf("device not connected: %+v", devs)
	}
}
