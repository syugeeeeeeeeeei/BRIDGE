package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/internal/yamlmini"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/products/cli/templates"
	productserver "github.com/syugeeeeeeeeeei/BRIDGE/src/products/server"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type serverFileConfig struct {
	SchemaVersion string `json:"schema_version"`
	Server        struct {
		Listen          string `json:"listen"`
		RequestTimeout  string `json:"request_timeout"`
		ShutdownTimeout string `json:"shutdown_timeout"`
	} `json:"server"`
	Limits struct {
		MaxRequestBytes       int64  `json:"max_request_bytes"`
		MaxConcurrentRequests int    `json:"max_concurrent_requests"`
		MaxNodes              int    `json:"max_nodes"`
		MaxEdges              int    `json:"max_edges"`
		MaxLogicalWorkers     int    `json:"max_logical_workers"`
		MaxWorkBudget         uint64 `json:"max_work_budget"`
	} `json:"limits"`
	Logging struct {
		Level  string `json:"level"`
		Format string `json:"format"`
	} `json:"logging"`
}

func serve(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return serveInit(args[1:], stdout, stderr)
		case "validate":
			return serveValidate(args[1:], stdout, stderr)
		case "show":
			return serveShow(args[1:], stdout, stderr)
		case "help", "--help", "-h":
			fmt.Fprintln(stdout, "Usage:\n  bridge serve [--config file]\n  bridge serve init [--comments full|summary|none]\n  bridge serve validate <file>\n  bridge serve show [--config file] [--resolved]")
			return 0
		}
	}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	configPath := fs.String("config", "", "server configuration file")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "error: unexpected positional arguments")
		return exitUsage
	}
	cfg := productserver.DefaultConfig()
	if *configPath != "" {
		loaded, err := loadServerConfig(*configPath)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitUsage
		}
		cfg = loaded
	}
	if err := applyServerEnvironment(&cfg); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	srv, err := productserver.New(cfg, logger)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintln(stderr, "BRIDGE server listening on", cfg.Listen)
	if err := srv.Run(ctx); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	return 0
}
func serveInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("output", "bridge-server.yaml", "output path; use - for stdout")
	fs.StringVar(out, "o", "bridge-server.yaml", "output path")
	levelText := fs.String("comments", "full", "full, summary, or none")
	overwrite := fs.Bool("overwrite", false, "overwrite existing file")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 0 {
		return exitUsage
	}
	level, err := templates.ParseLevel(*levelText)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitUsage
	}
	text := templates.Server(level)
	tmp, err := os.CreateTemp("", "bridge-server-config-*.yaml")
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	name := tmp.Name()
	if _, err = tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(name)
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	_, err = loadServerConfig(name)
	_ = os.Remove(name)
	if err != nil {
		fmt.Fprintln(stderr, "error: generated template is invalid:", err)
		return exitInternal
	}
	return writeTextOutput(stdout, stderr, *out, *overwrite, text)
}
func serveValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one config file is required")
		return exitUsage
	}
	cfg, err := loadServerConfig(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "invalid:", err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "valid: %s\n", cfg.Listen)
	return 0
}
func serveShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	path := fs.String("config", "", "server configuration file")
	resolved := fs.Bool("resolved", false, "show resolved values")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	cfg := productserver.DefaultConfig()
	var err error
	if *path != "" {
		cfg, err = loadServerConfig(*path)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitUsage
		}
	}
	if *resolved {
		if err := applyServerEnvironment(&cfg); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return exitUsage
		}
	}
	if err := json.NewEncoder(stdout).Encode(cfg); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return exitIO
	}
	return 0
}
func loadServerConfig(path string) (productserver.Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return productserver.Config{}, err
	}
	payload := b
	if ext := strings.ToLower(filepath.Ext(path)); ext == ".yaml" || ext == ".yml" {
		payload, err = yamlmini.ToJSON(b)
		if err != nil {
			return productserver.Config{}, err
		}
	}
	var f serverFileConfig
	if err := decodeStrict(payload, &f); err != nil {
		return productserver.Config{}, err
	}
	if f.SchemaVersion != "bridge.server.config.v1" {
		return productserver.Config{}, fmt.Errorf("schema_version must be bridge.server.config.v1")
	}
	cfg := productserver.DefaultConfig()
	if f.Server.Listen != "" {
		cfg.Listen = f.Server.Listen
	}
	if f.Server.RequestTimeout != "" {
		cfg.RequestTimeout, err = time.ParseDuration(f.Server.RequestTimeout)
		if err != nil {
			return cfg, fmt.Errorf("server.request_timeout: %w", err)
		}
	}
	if f.Server.ShutdownTimeout != "" {
		cfg.ShutdownTimeout, err = time.ParseDuration(f.Server.ShutdownTimeout)
		if err != nil {
			return cfg, fmt.Errorf("server.shutdown_timeout: %w", err)
		}
	}
	if f.Limits.MaxRequestBytes > 0 {
		cfg.MaxRequestBytes = f.Limits.MaxRequestBytes
	}
	if f.Limits.MaxConcurrentRequests > 0 {
		cfg.MaxConcurrentRequests = f.Limits.MaxConcurrentRequests
	}
	if f.Limits.MaxNodes > 0 {
		cfg.MaxNodes = f.Limits.MaxNodes
	}
	if f.Limits.MaxEdges > 0 {
		cfg.MaxEdges = f.Limits.MaxEdges
	}
	if f.Limits.MaxLogicalWorkers > 0 {
		cfg.MaxLogicalWorkers = f.Limits.MaxLogicalWorkers
	}
	if f.Limits.MaxWorkBudget > 0 {
		cfg.MaxWorkBudget = f.Limits.MaxWorkBudget
	}
	return cfg, cfg.Validate()
}

func applyServerEnvironment(cfg *productserver.Config) error {
	if cfg == nil {
		return errors.New("server config is nil")
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_SERVER_LISTEN")); v != "" {
		cfg.Listen = v
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_SERVER_REQUEST_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("BRIDGE_SERVER_REQUEST_TIMEOUT: %w", err)
		}
		cfg.RequestTimeout = d
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_SERVER_SHUTDOWN_TIMEOUT")); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("BRIDGE_SERVER_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = d
	}
	parseInt := func(name string, target *int) error {
		v := strings.TrimSpace(os.Getenv(name))
		if v == "" {
			return nil
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		*target = n
		return nil
	}
	if err := parseInt("BRIDGE_SERVER_MAX_CONCURRENT_REQUESTS", &cfg.MaxConcurrentRequests); err != nil {
		return err
	}
	if err := parseInt("BRIDGE_SERVER_MAX_NODES", &cfg.MaxNodes); err != nil {
		return err
	}
	if err := parseInt("BRIDGE_SERVER_MAX_EDGES", &cfg.MaxEdges); err != nil {
		return err
	}
	if err := parseInt("BRIDGE_SERVER_MAX_LOGICAL_WORKERS", &cfg.MaxLogicalWorkers); err != nil {
		return err
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_SERVER_MAX_REQUEST_BYTES")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("BRIDGE_SERVER_MAX_REQUEST_BYTES: %w", err)
		}
		cfg.MaxRequestBytes = n
	}
	if v := strings.TrimSpace(os.Getenv("BRIDGE_SERVER_MAX_WORK_BUDGET")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("BRIDGE_SERVER_MAX_WORK_BUDGET: %w", err)
		}
		cfg.MaxWorkBudget = n
	}
	return cfg.Validate()
}
