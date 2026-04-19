package main_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Radiushina/metrics-receiver.git/cmd/server"
	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
	"golang.org/x/net/context"
)

func TestNewMux_PostGauge_OK(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestNewMux_PostCounter_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestNewMux_InvalidType_BadRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
