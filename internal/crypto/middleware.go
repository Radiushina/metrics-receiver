package crypto

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
)

// DecryptRequest возвращает middleware: если тело помечено Content-Encryption,
// расшифровывает его приватным ключом и выставляет Content-Encoding: gzip
// (агент шифрует уже gzip-сжатое тело).
// Если priv == nil, middleware ничего не делает.
func DecryptRequest(priv *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if priv == nil || r.Header.Get(ContentEncryptionHeader) != ContentEncryptionValue {
				next.ServeHTTP(w, r)
				return
			}

			cipherBody, err := io.ReadAll(r.Body)
			_ = r.Body.Close()
			if err != nil {
				http.Error(w, "failed to read encrypted body", http.StatusBadRequest)
				return
			}

			plain, err := Decrypt(priv, cipherBody)
			if err != nil {
				http.Error(w, "failed to decrypt body", http.StatusBadRequest)
				return
			}

			r2 := r.Clone(r.Context())
			r2.Body = io.NopCloser(bytes.NewReader(plain))
			r2.ContentLength = int64(len(plain))
			r2.Header = r.Header.Clone()
			r2.Header.Del(ContentEncryptionHeader)
			// Агент шифрует gzip-тело без Content-Encoding на проводе.
			r2.Header.Set("Content-Encoding", "gzip")

			next.ServeHTTP(w, r2)
		})
	}
}
