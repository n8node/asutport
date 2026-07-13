package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func NewRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewAPIKeyRaw() (raw string, prefix string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = "asut_" + hex.EncodeToString(b)
	if len(raw) < 16 {
		return "", "", err
	}
	return raw, raw[:16], nil
}

func HashAPIKey(salt, rawKey string) string {
	sum := sha256.Sum256([]byte(salt + rawKey))
	return hex.EncodeToString(sum[:])
}
