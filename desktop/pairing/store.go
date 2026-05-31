package pairing

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Store interface {
	Load() ([]trustedDevice, error)
	Save([]trustedDevice) error
}

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			configDir = "."
		}
		path = filepath.Join(configDir, "voiXPe3per", "trusted-devices.json")
	}
	return &FileStore{path: path}
}

func (s *FileStore) Load() ([]trustedDevice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []trustedDevice{}, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return []trustedDevice{}, nil
	}

	var devices []trustedDevice
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *FileStore) Save(devices []trustedDevice) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}
