package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// sessionToken returns a stateless, HMAC-signed session value:
// timestamp.nonce.signature. No server-side session store is needed —
// validity is re-derived from the signature and timestamp alone.
func sessionToken(secret string) string {
	var nonce [16]byte
	_, _ = rand.Read(nonce[:])
	timestamp := strconv.FormatInt(time.Now().Unix(), 16)
	text := timestamp + "." + hex.EncodeToString(nonce[:])
	sig := hmac.New(sha256.New, []byte(secret))
	_, _ = sig.Write([]byte(text))
	return text + "." + hex.EncodeToString(sig.Sum(nil))
}

func validSession(secret, value string, maxAge time.Duration) bool {
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return false
	}

	age, ok := tokenAge(parts[0])
	if !ok || age > maxAge {
		return false
	}

	sig, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected := hmac.New(sha256.New, []byte(secret))
	_, _ = expected.Write([]byte(parts[0] + "." + parts[1]))
	return hmac.Equal(sig, expected.Sum(nil))
}

// sessionAge returns how long ago value was issued, without validating
// its signature — callers that need authenticity must call validSession
// first.
func sessionAge(value string) (time.Duration, bool) {
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return 0, false
	}
	return tokenAge(parts[0])
}

func tokenAge(hexTimestamp string) (time.Duration, bool) {
	ts, err := strconv.ParseInt(hexTimestamp, 16, 64)
	if err != nil {
		return 0, false
	}
	return time.Since(time.Unix(ts, 0)), true
}
