package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/gate"
	"github.com/syugeeeeeeeeeei/BRIDGE/src/buildinfo"
	"io"
	"mime"
	"net/http"
	"strings"
)

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"application_version": buildinfo.Version,
		"api_versions":        []string{"v1"},
		"features":            map[string]bool{"route": true, "serve": true, "benchmark": true, "trace": true, "http_transport": true, "local_process_transport": true},
		"schemas":             map[string][]string{"route_request": {gate.RouteRequestSchemaV1}, "route_response": {gate.RouteResultSchemaV1}, "error": {"bridge.error.v1"}},
	})
}
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	id := requestIDFrom(r.Context())
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "protocol", "Content-Typeはapplication/jsonでなければなりません", id)
		return
	}
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		writeError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "resource_limit", "同時実行上限に達しています", id)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxRequestBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req gate.RouteRequest
	if err := dec.Decode(&req); err != nil {
		code, status := "INVALID_JSON", http.StatusBadRequest
		if strings.Contains(err.Error(), "request body too large") {
			code, status = "REQUEST_TOO_LARGE", http.StatusRequestEntityTooLarge
		}
		writeError(w, status, code, "validation", err.Error(), id)
		return
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "validation", "複数のJSON値は指定できません", id)
		return
	}
	if req.Graph.Type != "inline" {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_ARGUMENT", "validation", "HTTP APIではgraph.type=inlineのみ利用できます", id)
		return
	}
	if len(req.Graph.Nodes) > s.cfg.MaxNodes || len(req.Graph.Edges) > s.cfg.MaxEdges || req.Route.Workers > s.cfg.MaxLogicalWorkers || (req.Budget.TotalWork != nil && *req.Budget.TotalWork > s.cfg.MaxWorkBudget) {
		writeError(w, http.StatusUnprocessableEntity, "RESOURCE_LIMIT_EXCEEDED", "resource_limit", "リクエストがサーバー上限を超えています", id)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.RequestTimeout)
	defer cancel()
	result, err := s.router.Route(ctx, req, gate.RouteOptions{})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			writeError(w, http.StatusRequestTimeout, "CANCELLED", "cancelled", "リクエストはキャンセルされました", id)
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			writeError(w, http.StatusGatewayTimeout, "DEADLINE_EXCEEDED", "timeout", "サーバーの処理期限を超過しました", id)
			return
		}
		var pe *gate.PublicError
		if errors.As(err, &pe) {
			writeError(w, http.StatusUnprocessableEntity, pe.Code, "validation", pe.Message, id)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal", "内部エラーが発生しました", id)
		return
	}
	w.Header().Set("X-Request-ID", id)
	writeJSON(w, http.StatusOK, result)
}
