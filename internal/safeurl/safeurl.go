// Package safeurl builds outbound HTTP clients that cannot be steered at
// internal infrastructure (SSRF). The check runs inside the dialer on the
// already-resolved IP, so DNS rebinding and redirect tricks do not bypass it.
package safeurl

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
	"172.16.0.0/12", "192.0.0.0/24", "192.168.0.0/16", "198.18.0.0/15",
	"224.0.0.0/4", "240.0.0.0/4", "::/128", "::1/128", "fc00::/7", "fe80::/10", "ff00::/8",
)

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(cidrs))
	for i, c := range cidrs {
		out[i] = netip.MustParsePrefix(c)
	}
	return out
}

// IsBlocked reports whether an IP is private, loopback, link-local, multicast or otherwise internal.
func IsBlocked(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	addr = addr.Unmap()
	for _, p := range blockedPrefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

type Options struct {
	Timeout      time.Duration
	MaxRedirects int
	// AllowPrivate disables the IP check. Tests only.
	AllowPrivate bool
	// AllowHost, when set, must approve the host of every redirect target.
	AllowHost func(host string) bool
}

var ErrBlocked = errors.New("destination address is not allowed")

// NewDialer returns a dialer that refuses internal destinations after DNS resolution.
func NewDialer(allowPrivate bool) *net.Dialer {
	return &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			if allowPrivate {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if IsBlocked(net.ParseIP(host)) {
				return fmt.Errorf("%w: %s", ErrBlocked, host)
			}
			return nil
		},
	}
}

func NewClient(o Options) *http.Client {
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.MaxRedirects == 0 {
		o.MaxRedirects = 3
	}
	dialer := NewDialer(o.AllowPrivate)
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = dialer.DialContext
	tr.Proxy = nil // never honour ambient proxy variables
	tr.MaxIdleConnsPerHost = 4
	return &http.Client{
		Timeout:   o.Timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= o.MaxRedirects {
				return errors.New("too many redirects")
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errors.New("redirect to non-http scheme")
			}
			if o.AllowHost != nil && !o.AllowHost(req.URL.Hostname()) {
				return fmt.Errorf("redirect to host %q is not allowed", req.URL.Hostname())
			}
			return nil
		},
	}
}

// ReadLimited reads at most max bytes and reports whether more data was available.
func ReadLimited(r io.Reader, max int64) ([]byte, bool, error) {
	b, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(b)) > max {
		return b[:max], true, nil
	}
	return b, false, nil
}
