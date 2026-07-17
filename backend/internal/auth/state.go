package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

// StateSigner produces and verifies the OAuth "state" CSRF token without any
// server-side storage: the nonce and expiry are carried in the value itself,
// authenticated with an HMAC derived from the JWT secret.
type StateSigner struct {
	secret []byte
	ttl    time.Duration
}

func NewStateSigner(jwtSecret string, ttl time.Duration) *StateSigner {
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte("oauth-state"))
	return &StateSigner{secret: mac.Sum(nil), ttl: ttl}
}

func (s *StateSigner) New() (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	nonceStr := base64.RawURLEncoding.EncodeToString(nonce)
	expiry := strconv.FormatInt(time.Now().Add(s.ttl).Unix(), 10)
	payload := nonceStr + "." + expiry
	return payload + "." + s.sign(payload), nil
}

func (s *StateSigner) Verify(state string) error {
	parts := strings.Split(state, ".")
	if len(parts) != 3 {
		return errors.New("malformed state")
	}

	payload := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(s.sign(payload)), []byte(parts[2])) {
		return errors.New("state signature mismatch")
	}

	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return errors.New("malformed state expiry")
	}
	if time.Now().Unix() > expiry {
		return errors.New("state expired")
	}

	return nil
}

func (s *StateSigner) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
