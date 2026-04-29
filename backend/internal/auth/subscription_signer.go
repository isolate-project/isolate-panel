package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SubscriptionSigner creates and verifies time-limited HMAC-SHA256 signed URLs.
type SubscriptionSigner struct {
	secret []byte
}

func NewSubscriptionSigner(secret string) *SubscriptionSigner {
	return &SubscriptionSigner{secret: []byte(secret)}
}

func (s *SubscriptionSigner) Sign(token string, ttl time.Duration) (sig string, exp int64) {
	exp = time.Now().Add(ttl).Unix()
	msg := fmt.Sprintf("%s:%d", token, exp)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil)), exp
}

func (s *SubscriptionSigner) Verify(token, sig string, exp int64) bool {
	if time.Now().Unix() > exp {
		return false
	}
	msg := fmt.Sprintf("%s:%d", token, exp)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func ParseSubscriptionQuery(rawSig, rawExp string) (sig string, exp int64, err error) {
	if rawSig == "" || rawExp == "" {
		return "", 0, fmt.Errorf("missing signature or expiration")
	}
	exp, err = strconv.ParseInt(rawExp, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid expiration")
	}
	return rawSig, exp, nil
}

func (s *SubscriptionSigner) SignedURL(token string, ttl time.Duration) string {
	sig, exp := s.Sign(token, ttl)
	return fmt.Sprintf("sig=%s&exp=%d", sig, exp)
}

func ExtractTokenFromPath(path string) string {
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if len(p) >= 32 {
			if _, err := base64.RawURLEncoding.DecodeString(p); err == nil {
				return p
			}
		}
	}
	return ""
}
