package web

import (
	"testing"
	"time"
)

func TestSessionTokenRoundTrip(t *testing.T) {
	token := sessionToken("s3cr3t")
	if !validSession("s3cr3t", token, time.Hour) {
		t.Error("validSession() = false for a freshly issued token, want true")
	}
}

func TestValidSessionRejectsWrongSecret(t *testing.T) {
	token := sessionToken("s3cr3t")
	if validSession("wrong-secret", token, time.Hour) {
		t.Error("validSession() = true with the wrong secret, want false")
	}
}

func TestValidSessionRejectsTamperedValue(t *testing.T) {
	token := sessionToken("s3cr3t")
	// Flip the last character to a value guaranteed to differ from it,
	// rather than a fixed replacement — the signature's last hex digit is
	// effectively random, so always substituting "0" would occasionally
	// (1 in 16) produce the exact same token and make this test flake.
	last := token[len(token)-1]
	replacement := byte('0')
	if last == replacement {
		replacement = '1'
	}
	tampered := token[:len(token)-1] + string(replacement)
	if validSession("s3cr3t", tampered, time.Hour) {
		t.Error("validSession() = true for a tampered token, want false")
	}
}

func TestValidSessionRejectsMalformedValue(t *testing.T) {
	for _, v := range []string{"", "a", "a.b", "a.b.c.d"} {
		if validSession("s3cr3t", v, time.Hour) {
			t.Errorf("validSession(%q) = true, want false", v)
		}
	}
}

func TestValidSessionRejectsExpiredToken(t *testing.T) {
	token := sessionToken("s3cr3t")
	if validSession("s3cr3t", token, -time.Second) {
		t.Error("validSession() = true for a token older than maxAge, want false")
	}
}

func TestSessionAge(t *testing.T) {
	token := sessionToken("s3cr3t")
	age, ok := sessionAge(token)
	if !ok {
		t.Fatal("sessionAge() ok = false, want true")
	}
	if age < 0 || age > time.Second {
		t.Errorf("sessionAge() = %v, want ~0", age)
	}
}

func TestSessionAgeRejectsMalformedValue(t *testing.T) {
	if _, ok := sessionAge("not-a-token"); ok {
		t.Error("sessionAge() ok = true for a malformed value, want false")
	}
}
