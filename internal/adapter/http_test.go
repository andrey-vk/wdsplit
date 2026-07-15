package adapter

import (
	"context"
	"net"
	"testing"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2001:4860:4860::8888", true}, // public IPv6 (Google DNS)

		{"127.0.0.1", false},       // loopback
		{"::1", false},             // loopback IPv6
		{"10.0.0.1", false},        // RFC1918 private
		{"172.16.0.1", false},      // RFC1918 private
		{"192.168.1.1", false},     // RFC1918 private
		{"169.254.169.254", false}, // link-local — the classic cloud metadata SSRF target
		{"224.0.0.1", false},       // multicast
		{"0.0.0.0", false},         // unspecified
		{"fc00::1", false},         // IPv6 unique local (private)
		{"fe80::1", false},         // IPv6 link-local
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("net.ParseIP(%q) = nil", tt.ip)
		}
		if got := isPublicIP(ip); got != tt.want {
			t.Errorf("isPublicIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestSafeDialContextRejectsLoopback(t *testing.T) {
	_, err := safeDialContext(context.Background(), "tcp", "127.0.0.1:80")
	if err == nil {
		t.Fatal("safeDialContext() error = nil, want error (loopback rejected)")
	}
}

func TestSafeDialContextRejectsCloudMetadataAddress(t *testing.T) {
	_, err := safeDialContext(context.Background(), "tcp", "169.254.169.254:80")
	if err == nil {
		t.Fatal("safeDialContext() error = nil, want error (link-local metadata address rejected)")
	}
}

func TestHostAllowed(t *testing.T) {
	allowed := []string{"example.com", "Other.Example.com"}

	if !hostAllowed("example.com", allowed) {
		t.Error("hostAllowed(example.com) = false, want true")
	}
	if !hostAllowed("OTHER.EXAMPLE.COM", allowed) {
		t.Error("hostAllowed should be case-insensitive")
	}
	if hostAllowed("evil.com", allowed) {
		t.Error("hostAllowed(evil.com) = true, want false")
	}
}
