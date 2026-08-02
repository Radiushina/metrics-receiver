// Package crypto реализует гибридное шифрование тел HTTP-запросов:
// AES-256-GCM для данных и RSA-OAEP для сессионного ключа.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// ContentEncryptionHeader — заголовок, которым агент помечает зашифрованное тело.
const ContentEncryptionHeader = "Content-Encryption"

// ContentEncryptionValue — значение заголовка для нашего формата (AES-GCM + RSA-OAEP).
const ContentEncryptionValue = "aesrsa"

// LoadPublicKey читает RSA public key из PEM-файла.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("public key: pem decode failed")
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("public key: not RSA")
		}
		return rsaPub, nil
	}

	rsaPub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("public key: parse: %w", err)
	}
	return rsaPub, nil
}

// LoadPrivateKey читает RSA private key из PEM-файла.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("private key: pem decode failed")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("private key: parse: %w", err)
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key: not RSA")
	}
	return rsaKey, nil
}

// Encrypt шифрует plaintext: AES-256-GCM + RSA-OAEP(SHA-256) для ключа.
// Формат: [2]len(encKey) | encKey | nonce | ciphertext+tag.
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	if pub == nil {
		return nil, errors.New("encrypt: nil public key")
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("encrypt: aes key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("encrypt: nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt: rsa: %w", err)
	}
	if len(encKey) > 0xffff {
		return nil, errors.New("encrypt: rsa ciphertext too large")
	}

	out := make([]byte, 2+len(encKey)+len(nonce)+len(ciphertext))
	binary.BigEndian.PutUint16(out[0:2], uint16(len(encKey)))
	offset := 2
	offset += copy(out[offset:], encKey)
	offset += copy(out[offset:], nonce)
	copy(out[offset:], ciphertext)
	return out, nil
}

// Decrypt расшифровывает данные, полученные Encrypt.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	if priv == nil {
		return nil, errors.New("decrypt: nil private key")
	}
	if len(data) < 2 {
		return nil, errors.New("decrypt: truncated payload")
	}

	encKeyLen := int(binary.BigEndian.Uint16(data[0:2]))
	offset := 2
	if encKeyLen <= 0 || len(data) < offset+encKeyLen {
		return nil, errors.New("decrypt: invalid enc key length")
	}
	encKey := data[offset : offset+encKeyLen]
	offset += encKeyLen

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: rsa: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < offset+nonceSize {
		return nil, errors.New("decrypt: truncated nonce")
	}
	nonce := data[offset : offset+nonceSize]
	ciphertext := data[offset+nonceSize:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: aes-gcm: %w", err)
	}
	return plain, nil
}
