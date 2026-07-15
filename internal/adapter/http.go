package adapter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// newSandboxHTTPClient builds an http.Client hardened against SSRF: every
// dial re-resolves the host and rejects non-public IPs, then connects to
// the validated IP directly (not the original hostname) so a second DNS
// lookup at connect time can't resolve somewhere else — otherwise a
// DNS-rebinding attacker could pass the check and still reach an internal
// address. Applies uniformly to the initial request and any redirects,
// since both go through the same DialContext.
func newSandboxHTTPClient(allowedHosts []string) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			if len(allowedHosts) > 0 && !hostAllowed(req.URL.Hostname(), allowedHosts) {
				return fmt.Errorf("redirect to disallowed host %q", req.URL.Hostname())
			}
			return nil
		},
		Transport: &http.Transport{
			DialContext: safeDialContext,
		},
	}
}

func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("split host/port: %w", err)
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	var target net.IP
	for _, ip := range ips {
		if isPublicIP(ip) {
			target = ip
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("refusing to connect to %s: no public address resolved", host)
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(target.String(), port))
}

func isPublicIP(ip net.IP) bool {
	return !ip.IsPrivate() &&
		!ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsMulticast() &&
		!ip.IsUnspecified()
}

func hostAllowed(host string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(host, a) {
			return true
		}
	}
	return false
}
