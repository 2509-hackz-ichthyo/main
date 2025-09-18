package config

import "testing"

func TestLoadWithEnv(t *testing.T) {
	t.Setenv(envServerPort, "8081")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerPort != "8081" {
		t.Fatalf("ServerPort = %q, want %q", cfg.ServerPort, "8081")
	}
}

func TestLoadDefault(t *testing.T) {
	t.Setenv(envServerPort, "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerPort != "3000" {
		t.Fatalf("ServerPort = %q, want %q", cfg.ServerPort, "3000")
	}
}
