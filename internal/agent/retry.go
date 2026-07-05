package agent

import (
	"errors"
	"net"
	"net/http"
	"syscall"
	"time"
)

// Интервалы перед 2-й, 3-й и 4-й попыткой (всего до четырёх попыток отправки).
var agentRetryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

var retriableTransportErrnos = []error{
	syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.EPIPE,
	syscall.ETIMEDOUT, syscall.EHOSTUNREACH, syscall.ENETUNREACH,
	syscall.ECONNABORTED, syscall.ENETDOWN,
}

// retrySleep подменяется в тестах, чтобы не ждать реальные секунды.
var retrySleep = func(d time.Duration) {
	time.Sleep(d)
}

func retryAgent(op func() (success bool, retriable bool, err error)) error {
	var lastErr error
	for attempt := 0; attempt < 1+len(agentRetryDelays); attempt++ {
		if attempt > 0 {
			retrySleep(agentRetryDelays[attempt-1])
		}
		ok, retry, err := op()
		if ok {
			return nil
		}
		lastErr = err
		if !retry {
			return err
		}
	}
	return lastErr
}

func isRetriableTransportErr(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && (dnsErr.IsTemporary || dnsErr.IsTimeout) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return isRetriableTransportErr(opErr.Err)
	}
	for _, target := range retriableTransportErrnos {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func isRetriableHTTPStatus(code int) bool {
	switch code {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
