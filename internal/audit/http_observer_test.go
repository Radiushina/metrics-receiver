package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

func newTestHTTPObserver(url string) *HTTPObserver {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryWaitMin = time.Millisecond
	retryClient.RetryWaitMax = 2 * time.Millisecond
	retryClient.RetryMax = 3

	return &HTTPObserver{
		url:    url,
		client: retryClient.StandardClient(),
		log:    zap.NewNop(),
	}
}

func TestHTTPObserver_Notify_RetriesOn503(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	obs := newTestHTTPObserver(srv.URL)
	err := obs.Notify(context.Background(), Event{
		TS:        1,
		Metrics:   []string{"HeapAlloc"},
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("ожидали 3 попытки, получили %d", got)
	}
}

func TestHTTPObserver_Notify_NoRetryOn400(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	obs := newTestHTTPObserver(srv.URL)
	err := obs.Notify(context.Background(), Event{TS: 1})
	if err == nil {
		t.Fatal("ожидали ошибку")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("ожидали 1 попытку без ретрая, получили %d", got)
	}
}
