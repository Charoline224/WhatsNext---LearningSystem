package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong password") {
		t.Fatal("wrong password verified")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	values := []string{"", "$argon2id$v=19$m=1,t=0,p=0$YQ$YQ", "$argon2i$v=19$m=65536,t=3,p=2$YQ$YQ"}
	for _, value := range values {
		if VerifyPassword(value, "password") {
			t.Fatalf("malformed hash verified: %q", value)
		}
	}
}
