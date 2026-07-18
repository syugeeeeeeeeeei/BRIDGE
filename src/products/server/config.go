package server

import (
	"errors"
	"strings"
	"time"
)

type Config struct {
	Listen                string
	RequestTimeout        time.Duration
	ShutdownTimeout       time.Duration
	MaxRequestBytes       int64
	MaxConcurrentRequests int
	MaxNodes              int
	MaxEdges              int
	MaxLogicalWorkers     int
	MaxWorkBudget         uint64
}

func DefaultConfig() Config {
	return Config{"127.0.0.1:8080", 30 * time.Second, 10 * time.Second, 16 << 20, 4, 1_000_000, 10_000_000, 16, 100_000_000}
}
func (c Config) Validate() error {
	if strings.TrimSpace(c.Listen) == "" {
		return errors.New("server.listen is required")
	}
	if c.RequestTimeout <= 0 || c.ShutdownTimeout <= 0 || c.MaxRequestBytes <= 0 || c.MaxConcurrentRequests <= 0 || c.MaxNodes <= 0 || c.MaxEdges <= 0 || c.MaxLogicalWorkers <= 0 || c.MaxWorkBudget == 0 {
		return errors.New("server limits and timeouts must be greater than zero")
	}
	return nil
}
