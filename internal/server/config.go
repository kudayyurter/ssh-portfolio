package server

import (
	"fmt"
	"strconv"
	"time"
)

// Config is everything the server reads from the environment.
type Config struct {
	ListenAddr  string
	HostKeyPath string
	PublicHost  string // shown in the "ssh -t <host>" hint
	MaxSessions int
	PerIP       int
	IdleTimeout time.Duration
	MaxSession  time.Duration

	// Connection-level limits, applied before any SSH handshake work.
	MaxConnsPerIP    int
	HandshakeTimeout time.Duration
}

// ConfigFromEnv reads the env vars listed in the spec, with defaults.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	c := Config{
		ListenAddr:  or(getenv("LISTEN_ADDR"), ":2222"),
		HostKeyPath: or(getenv("HOST_KEY_PATH"), "/data/ssh_host_ed25519"),
		PublicHost:  or(getenv("PUBLIC_HOST"), "term.kudayyurter.dev"),
		PerIP:       5,

		MaxConnsPerIP:    10,
		HandshakeTimeout: 15 * time.Second,
	}
	var err error
	if c.MaxSessions, err = strconv.Atoi(or(getenv("MAX_SESSIONS"), "100")); err != nil || c.MaxSessions < 1 {
		return Config{}, fmt.Errorf("MAX_SESSIONS: must be a positive integer")
	}
	if c.IdleTimeout, err = time.ParseDuration(or(getenv("IDLE_TIMEOUT"), "10m")); err != nil {
		return Config{}, fmt.Errorf("IDLE_TIMEOUT: %w", err)
	}
	if c.MaxSession, err = time.ParseDuration(or(getenv("MAX_SESSION"), "30m")); err != nil {
		return Config{}, fmt.Errorf("MAX_SESSION: %w", err)
	}
	return c, nil
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
