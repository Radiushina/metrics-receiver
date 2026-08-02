package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	plain := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	cipher, err := Encrypt(&priv.PublicKey, plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytesEqual(cipher, plain) {
		t.Fatal("ciphertext must differ from plaintext")
	}

	got, err := Decrypt(priv, cipher)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytesEqual(got, plain) {
		t.Fatalf("Decrypt: got %q, want %q", got, plain)
	}
}

func TestEncrypt_NilKey(t *testing.T) {
	if _, err := Encrypt(nil, []byte("x")); err == nil {
		t.Fatal("expected error for nil public key")
	}
}

func TestDecrypt_NilKey(t *testing.T) {
	if _, err := Decrypt(nil, []byte("x")); err == nil {
		t.Fatal("expected error for nil private key")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
