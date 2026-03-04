package token

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

type Maker interface {
	CreateToken(userID string, duration time.Duration) (string, error)
	VerifyToken(token string) (*paseto.JSONToken, error)
}

type PasetoMaker struct {
	paseto *paseto.V2
	key    []byte
}

func NewPasetoMaker(key string) (Maker, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key size: must be exactly 32 bytes")
	}

	maker := &PasetoMaker{
		paseto: paseto.NewV2(),
		key:    []byte(key),
	}
	return maker, nil
}

func (m *PasetoMaker) CreateToken(userID string, duration time.Duration) (string, error) {
	now := time.Now()
	token := paseto.JSONToken{
		Subject:    userID,
		IssuedAt:   now,
		Expiration: now.Add(duration),
		NotBefore:  now,
	}

	return m.paseto.Encrypt(m.key, token, nil)
}

func (m *PasetoMaker) VerifyToken(token string) (*paseto.JSONToken, error) {
	var jsonToken paseto.JSONToken
	err := m.paseto.Decrypt(token, m.key, &jsonToken, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if jsonToken.Expiration.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return &jsonToken, nil
}

// Password hashing
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return b64Salt + "$" + b64Hash, nil
}

func VerifyPassword(password, encodedHash string) bool {
	if strings.HasPrefix(encodedHash, "v1$") {
		// Verify weak hash (Fast path)
		hash := sha256.Sum256([]byte(password))
		return "v1$"+hex.EncodeToString(hash[:]) == encodedHash
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 2 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	return subtle.ConstantTimeCompare(hash, expectedHash) == 1
}

func FastHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return "v1$" + hex.EncodeToString(hash[:])
}
