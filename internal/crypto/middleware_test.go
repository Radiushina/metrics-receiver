package crypto_test

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	appcrypto "github.com/Radiushina/metrics-receiver.git/internal/crypto"
	"github.com/Radiushina/metrics-receiver.git/internal/middleware"
)

func TestDecryptRequest_MiddlewareRoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	plainJSON := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	var gzBuf bytes.Buffer
	zw := gzip.NewWriter(&gzBuf)
	if _, err := zw.Write(plainJSON); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	cipherBody, err := appcrypto.Encrypt(&priv.PublicKey, gzBuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	var gotBody []byte
	h := appcrypto.DecryptRequest(priv)(middleware.DecompressRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		gotBody = b
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(cipherBody))
	req.Header.Set(appcrypto.ContentEncryptionHeader, appcrypto.ContentEncryptionValue)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if !bytes.Equal(gotBody, plainJSON) {
		t.Fatalf("body after decrypt+gunzip: got %s want %s", gotBody, plainJSON)
	}
	var m map[string]any
	if err := json.Unmarshal(gotBody, &m); err != nil {
		t.Fatal(err)
	}
}

func TestDecryptRequest_NoopWithoutHeader(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	called := false
	h := appcrypto.DecryptRequest(priv)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if !called {
		t.Fatal("handler not called")
	}
}
