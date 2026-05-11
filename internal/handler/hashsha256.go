package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const hashSHA256Header = "HashSHA256"

func setResponseHashSHA256(w http.ResponseWriter, secretKey string, responseBody []byte) {
	key := strings.TrimSpace(secretKey)
	if key == "" {
		return
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(responseBody)
	w.Header().Set(hashSHA256Header, base64.StdEncoding.EncodeToString(mac.Sum(nil)))
}

// gzipBytesForHash matches encoding used by cmd/agent (gzip.DefaultCompression / NewWriter defaults).
func gzipBytesForHash(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(src); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hmacSHA256Sum(key string, msg []byte) []byte {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write(msg)
	return mac.Sum(nil)
}

// verifyRequestBodyHashSHA256 accepts either legacy HMAC(gzip(body)) or HMAC(body) for plain JSON clients.
func verifyRequestBodyHashSHA256(secretKey string, r *http.Request, body []byte) error {
	key := strings.TrimSpace(secretKey)
	if key == "" {
		return nil
	}
	raw := strings.TrimSpace(r.Header.Get(hashSHA256Header))
	if raw == "" {
		// Ключ на сервере включает подпись ответов; запрос без HashSHA256 — без проверки (совместимость с metricstest и старыми клиентами).
		return nil
	}
	got, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return errors.New("invalid HashSHA256")
	}
	if hmac.Equal(got, hmacSHA256Sum(key, body)) {
		return nil
	}
	gz, err := gzipBytesForHash(body)
	if err != nil {
		return errors.New("hash mismatch")
	}
	if hmac.Equal(got, hmacSHA256Sum(key, gz)) {
		return nil
	}
	return errors.New("hash mismatch")
}

func verifyPathUpdateRequestHash(r *http.Request, secretKey string) error {
	key := strings.TrimSpace(secretKey)
	if key == "" {
		return nil
	}
	raw := strings.TrimSpace(r.Header.Get(hashSHA256Header))
	if raw == "" {
		return nil
	}
	got, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return errors.New("invalid HashSHA256")
	}
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(r.URL.Path))
	want := mac.Sum(nil)
	if !hmac.Equal(got, want) {
		return errors.New("hash mismatch")
	}
	return nil
}

func writeProtectedPlainError(w http.ResponseWriter, secretKey string, code int, msg string) {
	if strings.TrimSpace(secretKey) == "" {
		http.Error(w, msg, code)
		return
	}
	body := []byte(msg + "\n")
	setResponseHashSHA256(w, secretKey, body)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func writeProtectedEmptyOK(w http.ResponseWriter, secretKey string) {
	if strings.TrimSpace(secretKey) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}
	setResponseHashSHA256(w, secretKey, nil)
	w.WriteHeader(http.StatusOK)
}
