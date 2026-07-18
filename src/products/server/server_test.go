package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthAndRoute(t *testing.T) {
	srv, err := New(DefaultConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != 200 {
		t.Fatalf("health=%d", rr.Code)
	}
	body := []byte(`{"schema_version":"bridge.route.request.v1","graph":{"type":"inline","nodes":[{"id":0},{"id":1}],"edges":[{"from":0,"to":1,"weight":1}]},"route":{"source":0,"target":1}}`)
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("route=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestServerRejectsFileGraph(t *testing.T) {
	srv, _ := New(DefaultConfig(), nil)
	body := []byte(`{"schema_version":"bridge.route.request.v1","graph":{"type":"file","path":"/etc/passwd"},"route":{"source":0,"target":1}}`)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/routes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != 422 {
		t.Fatalf("code=%d", rr.Code)
	}
}

func TestErrorContractV1(t *testing.T) {
	srv, _ := New(DefaultConfig(), nil)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/routes", bytes.NewBufferString(`{"schema_version":"bad","graph":{"type":"inline"},"route":{"source":0,"target":0}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "contract-test")
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Error         struct {
			Code      string `json:"code"`
			Category  string `json:"category"`
			Retryable bool   `json:"retryable"`
			RequestID string `json:"request_id"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.SchemaVersion != "bridge.error.v1" || payload.Error.Code != "INVALID_SCHEMA_VERSION" || payload.Error.Category != "validation" || payload.Error.Retryable || payload.Error.RequestID != "contract-test" {
		t.Fatalf("unexpected error contract: %+v", payload)
	}
}
