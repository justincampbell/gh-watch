// Package state persists watch snapshots between runs for --resume.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// File stores one watch's last snapshot, named by a hash of the watch's
// identity (repo, subcommand, target) so the same watch finds it again.
type File[S any] struct {
	Path string
	Key  string
}

type envelope[S any] struct {
	Key     string    `json:"key"`
	SavedAt time.Time `json:"saved_at"`
	State   *S        `json:"state"`
}

// New returns the state file for key under $XDG_STATE_HOME/gh-watch
// (default ~/.local/state/gh-watch).
func New[S any](key string) (*File[S], error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(key))
	return &File[S]{Path: filepath.Join(dir, hex.EncodeToString(sum[:8])+".json"), Key: key}, nil
}

func Dir() (string, error) {
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "gh-watch"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding state directory: %w", err)
	}
	return filepath.Join(home, ".local", "state", "gh-watch"), nil
}

// Load returns the saved snapshot, or nil if none exists.
func (f *File[S]) Load() (*S, error) {
	data, err := os.ReadFile(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var env envelope[S]
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("reading %s: %w", f.Path, err)
	}
	if env.Key != f.Key {
		return nil, nil
	}
	return env.State, nil
}

// Save writes the snapshot atomically.
func (f *File[S]) Save(s *S) error {
	data, err := json.Marshal(envelope[S]{Key: f.Key, SavedAt: time.Now(), State: s})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(f.Path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), f.Path)
}

// Clear removes the snapshot once a watch has finished.
func (f *File[S]) Clear() error {
	err := os.Remove(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
