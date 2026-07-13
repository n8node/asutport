package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 3
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
	minPassword  = 12
)

var weakPasswords = map[string]struct{}{
	"password1234": {}, "qwerty123456": {}, "asutport1234": {},
	"123456789012": {}, "changeme1234": {},
}

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", argonMemory, argonTime, argonThreads, b64Salt, b64Hash), nil
}

func CheckPassword(encoded, password string) bool {
	salt, hash, err := decodeArgon2Hash(encoded)
	if err != nil {
		return false
	}
	other := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return subtle.ConstantTimeCompare(hash, other) == 1
}

func ValidatePassword(password string) error {
	if len(password) < minPassword {
		return fmt.Errorf("password must be at least %d characters", minPassword)
	}
	if _, ok := weakPasswords[strings.ToLower(password)]; ok {
		return errors.New("password is too common")
	}
	return nil
}

func decodeArgon2Hash(encoded string) (salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, errors.New("invalid argon2 hash")
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, err
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, err
	}
	return salt, hash, nil
}
