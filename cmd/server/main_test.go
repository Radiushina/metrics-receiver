package main_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	main "github.com/Radiushina/metrics-receiver.git/cmd/server"
	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"go.uber.org/zap"
)

func TestNewMux_PostGauge_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge","value":12}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestNewMux_PostCounter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update", strings.NewReader(
		`{"id":"PollCount","type":"counter","delta":1}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestNewMux_InvalidType_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update", strings.NewReader(
		`{"id":"x","type":"unknown","value":1}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestNewMux_JSONEndpoints_TrailingSlash_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	upd, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update/", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge","value":1}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	upd.Header.Set("Content-Type", "application/json")
	updResp, err := http.DefaultClient.Do(upd)
	if err != nil {
		t.Fatal(err)
	}
	_ = updResp.Body.Close()
	if updResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /update/: status %d", updResp.StatusCode)
	}

	val, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/value/", strings.NewReader(
		`{"id":"HeapAlloc","type":"gauge"}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	val.Header.Set("Content-Type", "application/json")
	valResp, err := http.DefaultClient.Do(val)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = valResp.Body.Close() })
	if valResp.StatusCode != http.StatusOK {
		t.Fatalf("POST /value/: status %d", valResp.StatusCode)
	}
	if ct := valResp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type: %q", ct)
	}
}

func TestNewMux_GzipRequestBody_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	raw := []byte(`{"id":"HeapAlloc","type":"gauge","value":12.5}`)
	var gzBody bytes.Buffer
	zw := gzip.NewWriter(&gzBody)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update", bytes.NewReader(gzBody.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d, body %q", resp.StatusCode, string(b))
	}
}

func TestNewMux_GzipResponse_JSON_WhenAccepted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	upd, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/update", strings.NewReader(
		`{"id":"PollCount","type":"counter","delta":3}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	upd.Header.Set("Content-Type", "application/json")
	if _, err := http.DefaultClient.Do(upd); err != nil {
		t.Fatal(err)
	}

	val, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/value", strings.NewReader(
		`{"id":"PollCount","type":"counter"}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	val.Header.Set("Content-Type", "application/json")
	val.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(val)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ce := resp.Header.Get("Content-Encoding"); ce != "gzip" {
		t.Fatalf("Content-Encoding %q, want gzip", ce)
	}

	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zr.Close() })

	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	var got models.Metrics
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("invalid JSON: %v, body=%q", err, string(b))
	}
	if got.ID != models.PollCount || got.MType != models.Counter || got.Delta == nil || *got.Delta != 3 {
		t.Fatalf("response %+v", got)
	}
}

func TestNewMux_GzipResponse_HTML_WhenAccepted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ce := resp.Header.Get("Content-Encoding"); ce != "gzip" {
		t.Fatalf("Content-Encoding %q, want gzip", ce)
	}

	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = zr.Close() })
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "<h1>Metrics</h1>") {
		t.Fatalf("unexpected html body: %q", string(b))
	}
}

func TestNewMux_DoesNotGzipTextPlain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	svc.SetGauge("HeapAlloc", 42.5)
	h := handler.NewHandler(svc, nil, zap.NewNop(), nil)
	ts := httptest.NewServer(main.NewMux(zap.NewNop(), h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/value/gauge/HeapAlloc", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ce := resp.Header.Get("Content-Encoding"); ce != "" {
		t.Fatalf("Content-Encoding %q, want empty", ce)
	}
}
