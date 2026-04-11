package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type AlertConfig struct {
	Visual  bool   `json:"visual"`
	Desktop bool   `json:"desktop"`
	Webhook string `json:"webhook,omitempty"`
}

type MonitorConfig struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Interval int         `json:"interval"`
	Alerts   AlertConfig `json:"alerts"`
	// Custom check fields
	Command        string  `json:"command,omitempty"`
	ExpectedStatus int     `json:"expected_status,omitempty"`
	CPUThreshold   float64 `json:"cpu_threshold,omitempty"`
	MemThreshold   float64 `json:"mem_threshold,omitempty"`
}

type Store struct {
	path string
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Save(monitors []MonitorConfig) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create store dir: %w", err)
	}

	data, err := json.MarshalIndent(monitors, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal monitors: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write monitors: %w", err)
	}
	return nil
}

func (s *Store) Load() ([]MonitorConfig, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []MonitorConfig{}, nil
		}
		return nil, fmt.Errorf("read monitors: %w", err)
	}

	var monitors []MonitorConfig
	if err := json.Unmarshal(data, &monitors); err != nil {
		return nil, fmt.Errorf("unmarshal monitors: %w", err)
	}
	return monitors, nil
}
