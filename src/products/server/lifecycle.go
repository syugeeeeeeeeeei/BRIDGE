package server

import (
	"context"
	"errors"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

type Server struct {
	cfg       Config
	logger    *slog.Logger
	router    *gate.Router
	slots     chan struct{}
	http      *http.Server
	ready     atomic.Bool
	requestID atomic.Uint64
}

func New(cfg Config, logger *slog.Logger) (*Server, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{cfg: cfg, logger: logger, router: gate.NewRouter(), slots: make(chan struct{}, cfg.MaxConcurrentRequests)}
	s.ready.Store(true)
	s.http = &http.Server{Addr: cfg.Listen, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second}
	return s, nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if !s.ready.Load() {
			writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "unavailable", "サーバーは新規リクエストを受け付けていません", requestIDFrom(r.Context()))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	})
	mux.HandleFunc("GET /v1/capabilities", s.capabilities)
	mux.HandleFunc("POST /v1/routes", s.route)
	return s.requestContext(recoverer(mux, s.logger))
}
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- s.http.ListenAndServe() }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		s.ready.Store(false)
		sdctx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		return s.http.Shutdown(sdctx)
	}
}
