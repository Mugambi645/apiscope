# Architecture

```
Browser ──► Next.js (web) ──/api/*──► Gateway (Go) ──► catalog (JSON manifests, in memory)
                                         │   ├──► probe worker ──► public APIs (HTTPS)
                                         │   ├──► proxy ("Try it") ──► catalogued hosts only
                                         │   └──► runner ──docker run──► sandbox container
                                         │                                   │ HTTPS_PROXY
                                         │                              egress proxy ──► catalogued hosts :443
```

## Decisions

1. **Standard library only for the Go services.** No external modules means no supply-chain surface in the component that handles untrusted traffic and a build that works offline.
2. **`catalog.Store` and `search.Ranker` are interfaces.** Postgres and pgvector slot in behind them without touching handlers.
3. **SSRF is blocked in the dialer, not by string checks.** `safeurl` rejects private, loopback, link-local, CGNAT and multicast addresses on the resolved IP at connect time, so DNS rebinding and redirects cannot bypass it.
4. **The proxy is not an open proxy.** It only relays to hosts present in the catalog, strips credentials-bearing headers it should not forward, caps request and response sizes and drops `Set-Cookie`.
5. **Probes treat 401/403 as healthy for keyed APIs.** They only prove reachability, not that a key works.
6. **Sandbox containers have no direct internet.** They sit on an `internal` Docker network whose only neighbour is the egress proxy, which allows CONNECT to port 443 on catalog hosts.

## Known limits

- The runner mounts `/var/run/docker.sock`, which is root-equivalent on that host. Run it on a dedicated VM. gVisor (`--runtime=runsc`) or Firecracker is the next hardening step and is not included.
- Probe history is a bounded JSON file, not a time series.
- Doc previews use regular expressions, not an HTML parser, so unusual markup yields blank fields.
- Rate limiting is per process; run one gateway or move the limiter to Redis before scaling out.
- The egress proxy allows any path on a catalog host. That is intended: the sandbox may call the APIs it is demonstrating.
- Node's `NODE_USE_ENV_PROXY` is what routes `fetch` through the egress proxy in the JavaScript image (Node 24). If you change the image, verify it still honours the variable.
