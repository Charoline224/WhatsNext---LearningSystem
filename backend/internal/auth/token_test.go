package auth

import (
	"testing"
	"time"
)

func TestAccessTokenRoundTripAndExpiry(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	m := NewTokenManager("01234567890123456789012345678901", "whatsnext", 15*time.Minute, 30*24*time.Hour)
	m.now = func() time.Time { return now }
	token, expires, err := m.NewAccessToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if expires != 900 {
		t.Fatalf("expires=%d", expires)
	}
	claims, err := m.ParseAccessToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" {
		t.Fatalf("subject=%q", claims.Subject)
	}
	m.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err = m.ParseAccessToken(token); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestAccessTokenRejectsTampering(t *testing.T) {
	m := NewTokenManager("01234567890123456789012345678901", "whatsnext", time.Minute, time.Hour)
	token, _, _ := m.NewAccessToken("user-1")
	token = token[:len(token)-1] + "x"
	if _, err := m.ParseAccessToken(token); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestRefreshTokensAreRandomAndHashed(t *testing.T) {
	m := NewTokenManager("01234567890123456789012345678901", "whatsnext", time.Minute, time.Hour)
	one, hashOne, _, err := m.NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	two, _, _, _ := m.NewRefreshToken()
	if one == two {
		t.Fatal("refresh tokens repeated")
	}
	got := HashRefreshToken(one)
	if string(got) != string(hashOne) {
		t.Fatal("refresh token hash mismatch")
	}
}
