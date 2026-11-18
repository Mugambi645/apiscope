// Command egress is the only way out of the sandbox network. It is a minimal
// HTTP CONNECT proxy that allows port 443 on catalog hosts and nothing else.
package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"apiscope/internal/catalog"
	"apiscope/internal/safeurl"
)

func main() {
	apis, err := catalog.LoadDir(envOr("CATALOG_DIR", "catalog"))
	if err != nil {
		log.Fatalf("catalog: %v", err)
	}
	store := catalog.NewMemoryStore(apis)
	dialer := safeurl.NewDialer(false)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "only CONNECT is supported (https)", http.StatusMethodNotAllowed)
			return
		}
		host, port, err := net.SplitHostPort(r.Host)
		if err != nil || port != "443" || !store.HostAllowed(strings.ToLower(host)) {
			log.Printf("denied CONNECT %s", r.Host)
			http.Error(w, "destination not allowed", http.StatusForbidden)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		upstream, err := dialer.DialContext(ctx, "tcp", r.Host)
		if err != nil {
			http.Error(w, "upstream unreachable", http.StatusBadGateway)
			return
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			upstream.Close()
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		client, _, err := hj.Hijack()
		if err != nil {
			upstream.Close()
			return
		}
		_, _ = client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		go func() { _, _ = io.Copy(upstream, client); upstream.Close() }()
		go func() { _, _ = io.Copy(client, upstream); client.Close() }()
	})
	srv := &http.Server{Addr: ":" + envOr("PORT", "3128"), Handler: h, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("egress proxy listening on %s (%d allowed hosts)", srv.Addr, len(apis))
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
