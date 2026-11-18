# APIScope

A verified, runnable directory of open APIs. Every listing is probed on a schedule (uptime, latency, HTTPS, CORS), can be tried from the browser through a safe proxy, and can be run as JavaScript or Python in an isolated sandbox. Adding an API is a pull request with one JSON manifest.

Built from the *APIScope: Open-Source API Explorer Blueprint*. See `docs/ARCHITECTURE.md` for design decisions and `docs/APIScope-Tutorial.pdf` for the step-by-step guide.

## Quick start

```bash
# Everything except the code runner (needs Docker):
docker compose up --build            # http://localhost:3000

# Add the sandboxed code runner (JavaScript and Python):
docker compose --profile sandbox up --build
# The gateway uses http://runner:8081 automatically

# Or run locally without Docker:
make run                             # gateway on :8080
make web-dev                         # web on :3000
```

Tests: `make test` (Go vet and tests, web unit tests, TypeScript check).

## What is in this repo

| Path | Purpose |
| --- | --- |
| `cmd/gateway` | Public API: catalog, search, health, link previews, Try-it proxy, run relay |
| `cmd/runner` | Executes sandboxed code in short-lived Docker containers |
| `cmd/egress` | The sandbox's only way out: allowlisted HTTPS CONNECT proxy |
| `cmd/validate` | Manifest validator used by CI |
| `internal/*` | `catalog`, `search`, `safeurl`, `proxy`, `probe`, `preview`, `sandbox`, `ratelimit`, `httpapi` |
| `catalog/*.json` | One manifest per API (14 seeded) |
| `web/` | Next.js 14 app: search, API pages, playground |

## Add an API

Create `catalog/<slug>.json` and run `make validate`:

```json
{
  "slug": "open-meteo",
  "name": "Open-Meteo",
  "description": "Free weather forecast API with hourly and daily variables.",
  "categories": ["weather"],
  "auth": "none",
  "cors": true,
  "base_url": "https://api.open-meteo.com",
  "docs_url": "https://open-meteo.com/en/docs",
  "endpoints": [{
    "name": "Current weather", "method": "GET", "path": "/v1/forecast", "summary": "Current conditions",
    "params": [{"name": "latitude", "in": "query", "required": true, "example": "-1.29"}]
  }]
}
```

CI rejects invalid manifests, duplicate slugs, non-HTTPS URLs and unknown fields.

## Status against the blueprint

| Stage | State |
| --- | --- |
| 1. MVP: catalog, search, doc previews, Try-it, JS and Python playground | Built and tested |
| 2. Verification: probes, uptime badges, manifest PR flow | Built and tested |
| 3. Full playground: TypeScript, Go, Rust runners, gVisor/Firecracker | Not built; the snippets are generated, execution is JS and Python only |
| 4. AI layer: pgvector semantic search, "Ask this API" | Not built; `search.Ranker` is the seam |
| 5. Growth: stack builder, drift alerts, mobile app | Not built |

Deliberate deviations from the blueprint: manifests are JSON rather than YAML, and the catalog and probe history live in memory plus a JSON file rather than Postgres. Both were chosen so the whole service builds with the Go standard library; `catalog.Store` is the interface to swap.

## Security notes

The playground is the risky part. Read `docs/ARCHITECTURE.md` before deploying the runner: it mounts the Docker socket and must run on an isolated host.

## License

MIT
# apiscope
