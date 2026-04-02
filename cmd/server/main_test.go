package main_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Radiushina/metrics-receiver.git/cmd/server"
	"github.com/Radiushina/metrics-receiver.git/internal/handler"
	"github.com/Radiushina/metrics-receiver.git/internal/repository"
	"github.com/Radiushina/metrics-receiver.git/internal/service"
)

func TestNewMux_PostGauge_OK(t *testing.T) {
	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/update/gauge/HeapAlloc/12", nil)
	if err != nil {
		t.Fatal(err)
	}

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
	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/update/counter/PollCount/1", nil)
	if err != nil {
		t.Fatal(err)
	}

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
	repo := repository.NewRepository()
	svc := service.NewService(repo)
	h := handler.NewHandler(svc)
	ts := httptest.NewServer(main.NewMux(h))
	t.Cleanup(ts.Close)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/update/bad/x/1", nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
