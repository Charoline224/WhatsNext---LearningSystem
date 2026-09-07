package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}
type TokenManager struct {
	key        []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenManager(key, issuer string, accessTTL, refreshTTL time.Duration) *TokenManager {
	return &TokenManager{key: []byte(key), issuer: issuer, accessTTL: accessTTL, refreshTTL: refreshTTL, now: time.Now}
}

func (m *TokenManager) NewAccessToken(userID string) (string, int64, error) {
	now := m.now().UTC()
	claims := Claims{Subject: userID, Issuer: m.issuer, IssuedAt: now.Unix(), ExpiresAt: now.Add(m.accessTTL).Unix()}
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", 0, err
	}
	unsigned := rawURL(header) + "." + rawURL(payload)
	signature := m.sign(unsigned)
	return unsigned + "." + rawURL(signature), int64(m.accessTTL.Seconds()), nil
}

func (m *TokenManager) ParseAccessToken(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(hb, &header) != nil || header.Alg != "HS256" || header.Typ != "JWT" {
		return Claims{}, ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, m.sign(parts[0]+"."+parts[1])) {
		return Claims{}, ErrInvalidToken
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	var claims Claims
	if err != nil || json.Unmarshal(pb, &claims) != nil || claims.Subject == "" || claims.Issuer != m.issuer || claims.ExpiresAt <= m.now().Unix() || claims.IssuedAt > m.now().Add(time.Minute).Unix() {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (m *TokenManager) NewRefreshToken() (plain string, hash []byte, expiresAt time.Time, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], m.now().UTC().Add(m.refreshTTL), nil
}
func HashRefreshToken(token string) []byte { sum := sha256.Sum256([]byte(token)); return sum[:] }
func (m *TokenManager) sign(value string) []byte {
	mac := hmac.New(sha256.New, m.key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}
func rawURL(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
