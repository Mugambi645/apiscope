package safeurl

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsBlocked(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1": true, "10.1.2.3": true, "192.168.1.1": true, "172.20.0.5": true,
		"169.254.169.254": true, "100.64.0.1": true, "::1": true, "fd00::1": true,
		"::ffff:10.0.0.1": true, "8.8.8.8": false, "1.1.1.1": false, "2606:4700::1111": false,
	}
	for ip, want := range cases {
		if got := IsBlocked(net.ParseIP(ip)); got != want {
			t.Errorf("IsBlocked(%s) = %v, want %v", ip, got, want)
		}
	}
}

func TestClientBlocksLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("secret")) }))
	defer srv.Close()
	_, err := NewClient(Options{}).Get(srv.URL)
	if err == nil || !errors.Is(err, ErrBlocked) && !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected SSRF block, got %v", err)
	}
	resp, err := NewClient(Options{AllowPrivate: true}).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestRedirectHostAllowlist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example/", http.StatusFound)
	}))
	defer srv.Close()
	c := NewClient(Options{AllowPrivate: true, AllowHost: func(h string) bool { return h == "127.0.0.1" }})
	if _, err := c.Get(srv.URL); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected redirect to be rejected, got %v", err)
	}
}

func TestReadLimited(t *testing.T) {
	b, trunc, _ := ReadLimited(strings.NewReader("abcdef"), 4)
	if string(b) != "abcd" || !trunc {
		t.Fatalf("got %q %v", b, trunc)
	}
	b, trunc, _ = ReadLimited(strings.NewReader("abc"), 4)
	if string(b) != "abc" || trunc {
		t.Fatalf("got %q %v", b, trunc)
	}
}
