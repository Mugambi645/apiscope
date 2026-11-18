"""Story for the APIScope tutorial. Code excerpts are read from the working tree."""
from reportlab.platypus import PageBreak, Spacer, CondPageBreak
from components import (H1, H2, H3, P, small, bullets, source, code_block, callout, table,
                        architecture_diagram)
import theme as T
from reportlab.platypus import Paragraph
import evidence


def build_story(version):
    s = []
    go = evidence.go_results()
    web = evidence.web_results()
    go_src, go_test, web_loc = evidence.loc()
    go_pass = sum(c["pass"] for c in go.values())
    go_fail = sum(c["fail"] for c in go.values())

    # ---------------------------------------------------------------- 1
    s += [H1("1. What you are building"),
          P("APIScope is a directory of open APIs where every listing is <b>verified</b> and "
            "<b>runnable</b>. Most API lists are static text. APIScope probes each API on a schedule, shows "
            "its measured uptime and latency, lets you send a real request from the browser, and runs generated "
            "JavaScript or Python in an isolated sandbox."),
          P("This guide walks through the code in the repository, explains each design decision, and shows how to "
            "run, test, extend and deploy it. It follows the <i>APIScope: Open-Source API Explorer Blueprint</i>."),
          H2("Blueprint features and where they stand"),
          table(["Blueprint feature", "State", "Where"],
                [["Verified, not listed (health probes, badges)", "Built", "<font face='Courier'>internal/probe</font>"],
                 ["Playground that runs code", "Built for JavaScript and Python", "<font face='Courier'>internal/sandbox</font>, <font face='Courier'>cmd/runner</font>"],
                 ["Code for Go, TypeScript, Rust", "Snippets generated; execution not built", "<font face='Courier'>web/lib/codegen.ts</font>"],
                 ["Documentation link previews", "Built", "<font face='Courier'>internal/preview</font>"],
                 ["Regional lens (Kenya, Africa)", "Built as region tags and a filter; small seed set", "<font face='Courier'>catalog/*.json</font>"],
                 ["Open-source contribution by manifest", "Built (JSON manifest, CI validator)", "<font face='Courier'>cmd/validate</font>"],
                 ["Semantic search (pgvector)", "Not built; lexical ranking with a swap point", "<font face='Courier'>internal/search</font>"],
                 ["Ask this API (RAG)", "Not built", "Roadmap, section 16"],
                 ["Stack builder, drift alerts, mobile app", "Not built", "Roadmap, section 16"]],
                [190, 150, 146]),
          callout("Two deliberate deviations from the blueprint",
                  "Manifests are <b>JSON</b>, not YAML, and the catalog and probe history live in memory plus a JSON "
                  "file, not Postgres. The Go services use only the standard library, so there is no third-party "
                  "code in the component that handles untrusted traffic. <font face='Courier'>catalog.Store</font> "
                  "and <font face='Courier'>search.Ranker</font> are the interfaces where Postgres and pgvector plug in."),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 2
    s += [H1("2. Architecture"),
          P("Requests flow through four tiers. The browser only ever talks to the Next.js app; Next.js forwards "
            "<font face='Courier'>/api/*</font> to the Go gateway, so the gateway needs no public CORS configuration. "
            "The amber boxes are the sandbox path, the riskiest part of the system."),
          architecture_diagram(),
          Paragraph("Figure 1. Runtime architecture. Amber marks the untrusted-code path.", T.caption),
          H2("Design decisions"),
          table(["Decision", "Reason"],
                [["Standard library only for Go", "No dependency surface in the service that touches untrusted input; builds offline."],
                 ["SSRF blocked in the dialer", "Checks the resolved IP at connect time, so DNS rebinding and redirects cannot bypass it."],
                 ["Proxy relays to catalog hosts only", "A general-purpose proxy would be an open relay. The catalog is the allowlist."],
                 ["Keyed APIs: 401/403 counts as healthy", "A probe has no key; it can prove reachability, not that a key works."],
                 ["Sandbox on an internal Docker network", "Containers have no route out except the egress proxy."],
                 ["Stores behind interfaces", "Postgres and pgvector replace the in-memory store and lexical ranker without touching handlers."]],
                [170, 316]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 3
    s += [H1("3. Prerequisites and setup"),
          table(["Tool", "Version used", "Needed for"],
                [["Go", "1.22 (built and tested on 1.22.2)", "Gateway, runner, egress, validator"],
                 ["Node.js", "22 (web), 24 in the sandbox image", "Next.js app; JavaScript sandbox"],
                 ["Docker + Compose", "Any recent version", "Containers and the code runner"],
                 ["Python 3 + ReportLab", "ReportLab 5.x", "Regenerating this PDF (optional)"]],
                [110, 190, 186]),
          H2("Get the code and verify your toolchain"),
          code_block("git clone https://github.com/mugambi645/apiscope.git\ncd apiscope\n"
                     "go version          # go1.22 or newer\nnode -v             # v22 or newer\n"
                     "make test           # go vet, go test, web tests, typecheck", "shell"),
          callout("Repository location",
                  "The commands assume the project is pushed to your GitHub account. Replace the URL if you host it "
                  "elsewhere. Nothing in the code depends on the URL except the probe User-Agent string."),
          H2("Run it"),
          H3("Option A: Docker Compose (web + gateway)"),
          code_block("docker compose up --build\n# open http://localhost:3000", "shell"),
          H3("Option B: Locally without Docker"),
          code_block("make run        # terminal 1: gateway on :8080\nmake web-dev    # terminal 2: web on :3000", "shell"),
          H3("Option C: With the sandboxed code runner"),
          code_block("# .env\nRUNNER_URL=http://runner:8081\n\ndocker compose --profile sandbox up --build", "shell"),
          callout("Read section 11 first", "The runner mounts the Docker socket. Do not run it on a machine you "
                  "care about until you have read the security notes.", "warn"),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 4
    s += [H1("4. Repository tour"),
          table(["Path", "Purpose"],
                [["<font face='Courier'>cmd/gateway</font>", "Public HTTP API: catalog, search, health, previews, Try-it proxy, run relay"],
                 ["<font face='Courier'>cmd/runner</font>", "Runs sandboxed code in short-lived Docker containers"],
                 ["<font face='Courier'>cmd/egress</font>", "HTTPS CONNECT proxy; the sandbox's only way out"],
                 ["<font face='Courier'>cmd/validate</font>", "Validates every manifest; used by CI"],
                 ["<font face='Courier'>internal/catalog</font>", "Manifest model, validation, loader, <font face='Courier'>Store</font> interface"],
                 ["<font face='Courier'>internal/search</font>", "Filters and lexical ranking behind <font face='Courier'>Ranker</font>"],
                 ["<font face='Courier'>internal/safeurl</font>", "SSRF-safe HTTP client and dialer"],
                 ["<font face='Courier'>internal/proxy</font>", "Try-it relay and its policy"],
                 ["<font face='Courier'>internal/probe</font>", "Health checks, ring buffer, persistence"],
                 ["<font face='Courier'>internal/preview</font>", "Documentation link unfurling with cache"],
                 ["<font face='Courier'>internal/sandbox</font>", "Docker command builder and execution limits"],
                 ["<font face='Courier'>internal/ratelimit</font>", "Per-IP token bucket"],
                 ["<font face='Courier'>internal/httpapi</font>", "Routes, CORS, security headers"],
                 ["<font face='Courier'>catalog/*.json</font>", "14 seeded API manifests"],
                 ["<font face='Courier'>web/</font>", "Next.js 14 app: search, API page, playground, code generator"]],
                [150, 336]),
          P(f"The Go source is about <b>{go_src:,}</b> lines plus <b>{go_test:,}</b> lines of tests; the web app is "
            f"about <b>{web_loc:,}</b> lines of TypeScript and CSS. Counts were measured when this PDF was built."),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 5
    s += [H1("5. The catalog"),
          P("Every API is one JSON file in <font face='Courier'>catalog/</font>. The gateway loads and validates all "
            "of them at boot and refuses to start if any is invalid, so bad data never reaches production. "
            "Decoding uses <font face='Courier'>DisallowUnknownFields</font>, which catches typos in field names."),
          source("catalog/frankfurter.json", '"categories"', 24, "catalog/frankfurter.json (excerpt, from categories onward)"),
          H2("Validation reports every problem at once"),
          P("Contributors fix a manifest in one pass because <font face='Courier'>Validate</font> collects errors "
            "with <font face='Courier'>errors.Join</font> instead of stopping at the first."),
          source("internal/catalog/catalog.go", "func (a *API) Validate() error", 26),
          bullets(["Slugs are lowercase kebab-case, so they are safe in URLs and file names.",
                   "<font face='Courier'>base_url</font>, <font face='Courier'>docs_url</font> and <font face='Courier'>spec_url</font> must be HTTPS.",
                   "Every <font face='Courier'>{placeholder}</font> in a path needs a matching path parameter.",
                   "Required parameters must carry an example, so the playground can always build a working request."]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 6
    s += [H1("6. Search and filters"),
          P("The first ranker is lexical. Every query term must match somewhere; name matches outrank tag, region "
            "and category matches, which outrank description matches. Filters narrow by category, auth type, "
            "browser-CORS support and region."),
          source("internal/search/search.go", "for _, t := range qt {", 20),
          P("Because the gateway depends on the <font face='Courier'>Ranker</font> interface, semantic search is an "
            "additive change: implement the same method with pgvector embeddings and combine its score with these "
            "filters."),
          H2("Public API"),
          table(["Endpoint", "Purpose"],
                [["<font face='Courier'>GET /api/apis?q=&amp;category=&amp;auth=&amp;cors=&amp;region=</font>", "Search with filters; each item includes its health summary"],
                 ["<font face='Courier'>GET /api/apis/{slug}</font>", "One API with endpoints and health"],
                 ["<font face='Courier'>GET /api/apis/{slug}/health</font>", "Probe history (last 100 checks)"],
                 ["<font face='Courier'>GET /api/apis/{slug}/preview</font>", "Documentation link preview"],
                 ["<font face='Courier'>GET /api/categories</font>", "Categories with counts"],
                 ["<font face='Courier'>POST /api/proxy</font>", "Try-it relay (rate limited)"],
                 ["<font face='Courier'>POST /api/run</font>", "Sandbox execution via the runner (rate limited)"]],
                [270, 216]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 7
    s += [H1("7. SSRF-safe networking"),
          P("Anything that fetches a URL on a user's behalf can be steered at internal services such as cloud "
            "metadata endpoints. Checking the hostname string is not enough, because DNS can return a private "
            "address, or a public site can redirect to one."),
          P("<font face='Courier'>safeurl</font> checks the <b>resolved IP inside the dialer's Control hook</b>, "
            "after DNS and for every redirect hop. The block list covers loopback, private ranges, link-local "
            "(including 169.254.169.254), carrier-grade NAT and multicast."),
          source("internal/safeurl/safeurl.go", "var blockedPrefixes", 6),
          source("internal/safeurl/safeurl.go", "func NewDialer", 20),
          P("The client also ignores ambient proxy variables and caps redirects. The proxy additionally pins "
            "redirects to catalog hosts through <font face='Courier'>Options.AllowHost</font>. Tests prove a loopback "
            "server is refused and that a redirect to an unlisted host is rejected."),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 8
    s += [H1("8. The Try-it proxy"),
          P("The proxy exists because browsers cannot call many APIs directly (no CORS) and because the page "
            "should show status and latency. It is deliberately narrow."),
          source("internal/proxy/proxy.go", "func (h *Handler) Validate", 28),
          bullets(["Only hosts present in the catalog are reachable; <font face='Courier'>evil.example</font> is rejected before any network call.",
                   "Only GET and POST; request body capped at 64 KB; response capped at 1 MB and flagged as truncated.",
                   "Cookies, <font face='Courier'>Host</font>, forwarding and <font face='Courier'>Sec-*</font> headers cannot be set by the caller.",
                   "<font face='Courier'>Set-Cookie</font> is never returned to the browser; only a short list of response headers is exposed.",
                   "API keys come from the browser per request. The server does not store or log them."]),
          callout("What this does not protect against",
                  "A catalog host can itself respond with anything. The proxy returns it as text for display and never "
                  "executes it. If you accept community manifests, review new hosts before merging, because merging one "
                  "adds it to the proxy and egress allowlists.", "warn"),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 9
    s += [H1("9. Health probes"),
          P("A background worker probes each API's first GET endpoint using its example values, on a schedule "
            "(default 30 minutes, four at a time). Each result records status, latency, HTTPS and whether the "
            "response allowed cross-origin use. The last 100 results per API are kept and saved atomically "
            "(write to a temp file, then rename)."),
          source("internal/probe/probe.go", "// Keyed APIs answer", 8),
          P("Summaries computed from the history power the badge on every card: uptime percentage, median "
            "latency of successful checks, time of the last check, and whether CORS was actually observed."),
          callout("Declared versus verified",
                  "Manifests declare <font face='Courier'>cors</font>. The probe sends an "
                  "<font face='Courier'>Origin</font> header and records whether the response allowed it. The API page "
                  "shows \"verified\" only when a probe saw it, and \"declared\" otherwise."),
          H2("Documentation link previews"),
          P("<font face='Courier'>preview.Service</font> fetches each API's catalogued docs URL, extracts the "
            "title, description and favicon (Open Graph tags first), caches for 24 hours and caps the download at "
            "256 KB. It only fetches URLs from the catalog, never a URL typed by a visitor."),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 10
    s += [H1("10. The web app"),
          P("The Next.js app has two screens. The home page is a client component with debounced search and filter "
            "controls. The API page is a server component, so each API has crawlable HTML, and it hosts the "
            "client-side playground."),
          H2("Code generation"),
          P("<font face='Courier'>buildRequest</font> turns an endpoint plus the values the user typed into a concrete "
            "request. <font face='Courier'>generate</font> then renders it as cURL, JavaScript, TypeScript, Python, Go "
            "or Rust. Values fall back to each parameter's example, so the default snippet always works."),
          source("web/lib/codegen.ts", "export function buildRequest", 18),
          P("Generated JavaScript and Python were syntax-checked with <font face='Courier'>node --check</font> and "
            "<font face='Courier'>py_compile</font>, and the Go output with <font face='Courier'>gofmt -e</font>."),
          H2("The playground"),
          bullets(["<b>Try it</b> posts the request to <font face='Courier'>/api/proxy</font> and shows status, latency and pretty-printed JSON.",
                   "Language tabs show editable code. Edits are kept until you switch endpoints or press Reset.",
                   "<b>Run in sandbox</b> appears for JavaScript and Python and posts to <font face='Courier'>/api/run</font>.",
                   "For keyed APIs a password field takes the key; it lives only in page state."]),
          source("web/components/Playground.tsx", "async function tryIt()", 8),
          callout("Build-time gateway URL",
                  "<font face='Courier'>next.config.mjs</font> bakes the <font face='Courier'>/api</font> rewrite target "
                  "into the build. Set <font face='Courier'>GATEWAY_URL</font> when you <b>build</b>, not only when you "
                  "start. The Dockerfile takes it as a build argument. This was found during testing: a build made "
                  "with the default value returned a 500 on <font face='Courier'>/api/*</font> until it was rebuilt.", "warn"),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 11
    s += [H1("11. The code sandbox"),
          P("The playground runs code that strangers type. The design assumes the code is hostile and limits "
            "what it can do rather than trying to detect bad code."),
          H2("Isolation policy"),
          P("All policy lives in one pure function, <font face='Courier'>Config.Args</font>, which builds the "
            "<font face='Courier'>docker run</font> command. Being pure, it is unit tested without Docker."),
          source("internal/sandbox/sandbox.go", "func (c Config) Args", 24),
          table(["Control", "Effect"],
                [["<font face='Courier'>--read-only</font>, tmpfs <font face='Courier'>/tmp</font> (noexec)", "Code cannot write to the image; scratch space is small and non-executable"],
                 ["<font face='Courier'>--cap-drop ALL</font>, <font face='Courier'>no-new-privileges</font>", "No Linux capabilities; no privilege escalation"],
                 ["<font face='Courier'>--user 65534:65534</font>", "Runs as nobody"],
                 ["<font face='Courier'>--memory 128m --cpus 0.5 --pids-limit 64</font>", "Bounds memory, CPU and fork bombs"],
                 ["10 s timeout, then <font face='Courier'>docker kill</font> by name", "Killing the CLI alone would leave the container running"],
                 ["64 KB output cap, 20 KB code cap, 4 concurrent jobs", "Bounds memory use in the runner and returns 429 when busy"],
                 ["Internal network + egress proxy", "No direct internet; only CONNECT :443 to catalog hosts"]],
                [210, 276]),
          H2("Network path"),
          P("The <font face='Courier'>apiscope_sandbox</font> network is declared <font face='Courier'>internal: true</font>, "
            "so containers on it cannot reach the internet. The egress service sits on that network and on the default "
            "one. Sandbox code reaches APIs through <font face='Courier'>HTTPS_PROXY</font>; Python's urllib honours it "
            "and Node 24 honours it when <font face='Courier'>NODE_USE_ENV_PROXY=1</font> is set."),
          callout("The Docker socket is the weakest link",
                  "The runner container mounts <font face='Courier'>/var/run/docker.sock</font>. Anyone who compromises the "
                  "runner process controls the host's Docker daemon, which is equivalent to root on that machine. Run the "
                  "runner on a dedicated VM with nothing else on it. The next hardening step is a rootless or gVisor "
                  "runtime (<font face='Courier'>--runtime=runsc</font>) or Firecracker. Neither is included.", "danger"),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 12
    s += [H1("12. Testing: what was actually verified"),
          P("These figures were produced by running the suites while this PDF was generated."),
          table(["Package", "Tests passed", "What they prove"],
                [[f"<font face='Courier'>{pkg}</font>", str(c['pass']), why] for pkg, c, why in [
                    ("internal/catalog", go.get("internal/catalog", {}) or {"pass": 0}, "Validation collects all errors; duplicates and unknown fields rejected; the 14 seeded manifests load"),
                    ("internal/search", go.get("internal/search", {}) or {"pass": 0}, "Ranking order, filters, all-terms-must-match, region searchable"),
                    ("internal/safeurl", go.get("internal/safeurl", {}) or {"pass": 0}, "Private ranges blocked; loopback refused; redirect allowlist; size cap"),
                    ("internal/proxy", go.get("internal/proxy", {}) or {"pass": 0}, "Host allowlist, header and method policy, truncation, no Set-Cookie leak"),
                    ("internal/probe", go.get("internal/probe", {}) or {"pass": 0}, "Health semantics for keyed vs open APIs, summary maths, persistence, ring cap"),
                    ("internal/preview", go.get("internal/preview", {}) or {"pass": 0}, "Title, description and favicon extraction with fallbacks"),
                    ("internal/sandbox", go.get("internal/sandbox", {}) or {"pass": 0}, "Isolation flags present; output and exit code; timeout kill; size and busy limits (fake docker)"),
                    ("internal/ratelimit", go.get("internal/ratelimit", {}) or {"pass": 0}, "Burst, refill, key independence, eviction"),
                    ("internal/httpapi", go.get("internal/httpapi", {}) or {"pass": 0}, "Search filters over HTTP, 404, CORS header, run disabled, rate limiting")]],
                [120, 62, 304]),
          P(f"<b>Go:</b> {go_pass} tests passed, {go_fail} failed, run with the race detector. "
            f"<b>Web:</b> {web.get('pass', 0)} of {web.get('tests', 0)} unit tests passed "
            "(code generation), TypeScript typecheck clean, production build succeeds."),
          H2("Smoke test"),
          bullets(["Gateway and web started together: <font face='Courier'>/api/apis?q=currency</font> through the Next.js rewrite returned the two currency APIs.",
                   "<font face='Courier'>POST /api/proxy</font> to an unlisted host returned a policy error, and a plain-HTTP metadata address was refused by the https-only rule (the IP block itself is covered by the safeurl unit tests, not this smoke test).",
                   "<font face='Courier'>POST /api/run</font> without a runner returned 503 as designed.",
                   "The API detail page rendered the playground; an unknown slug returned 404."]),
          callout("Not verified in the build environment",
                  "The environment had no Docker and no outbound internet. So these were <b>not</b> exercised: "
                  "<font face='Courier'>docker compose up</font>, real sandbox execution, the egress proxy against a live "
                  "API, live probes, live doc previews, Node 24's proxy behaviour, and the CI steps for gosec and cosign. "
                  "Run section 3's Option C on a throwaway VM and the checklist in section 17 before relying on them.", "warn"),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 13
    s += [H1("13. Adding an API"),
          P("Contributing is a pull request that adds one file."),
          bullets(["Create <font face='Courier'>catalog/&lt;slug&gt;.json</font> using an existing manifest as a template.",
                   "Give every required parameter an example value that works today.",
                   "Run <font face='Courier'>make validate</font>. CI runs the same command and blocks the merge on errors.",
                   "Open the pull request. After merge and deploy, the first probe produces the badge."]),
          code_block("make validate\n# OK: 14 manifests valid\n\n# a bad manifest reports everything at once:\n"
                     "# INVALID: x.json: slug \"Bad Slug\" must be lowercase kebab-case\n"
                     "# auth must be none, apikey or oauth (got \"magic\")", "shell"),
          H2("Review checklist for maintainers"),
          bullets(["Is the host one you are comfortable adding to the proxy and sandbox egress allowlists?",
                   "Is the licence compatible with listing, and do the terms allow programmatic probing?",
                   "Does the example request avoid personal data and avoid being expensive for the provider?",
                   "Is <font face='Courier'>auth</font> accurate? Probes treat keyed APIs as healthy on 401/403."]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 14
    s += [H1("14. Deploying"),
          H2("Shape"),
          bullets(["Start with one small VPS (or Fly.io) running web and gateway.",
                   "Put the <b>runner and egress on a separate VM</b> that holds nothing else, and point "
                   "<font face='Courier'>RUNNER_URL</font> at it over a private network or an authenticated tunnel.",
                   "Terminate TLS at a reverse proxy. Set <font face='Courier'>TRUST_PROXY=true</font> only when that "
                   "proxy sets <font face='Courier'>X-Forwarded-For</font>, otherwise clients can spoof their IP and "
                   "dodge rate limits."]),
          H2("Configuration"),
          table(["Variable", "Default", "Meaning"],
                [["<font face='Courier'>PORT</font>", "8080", "Gateway listen port"],
                 ["<font face='Courier'>CATALOG_DIR</font>", "catalog", "Manifest directory"],
                 ["<font face='Courier'>PROBES_ENABLED</font>", "true", "Run the probe worker"],
                 ["<font face='Courier'>PROBE_INTERVAL_MIN</font>", "30", "Minutes between probe rounds"],
                 ["<font face='Courier'>PROBE_FILE</font>", "data/probes.json", "Persisted probe history"],
                 ["<font face='Courier'>READ_RPM</font> / <font face='Courier'>TRY_RPM</font>", "240 / 30", "Per-IP requests per minute"],
                 ["<font face='Courier'>RUNNER_URL</font>", "(empty)", "Enables <font face='Courier'>/api/run</font>"],
                 ["<font face='Courier'>TRUST_PROXY</font>", "false", "Honour X-Forwarded-For"],
                 ["<font face='Courier'>GATEWAY_URL</font>", "http://localhost:8080", "Web build argument"]],
                [150, 110, 226]),
          H2("Continuous integration"),
          P("The workflow runs vet, race-enabled tests, manifest validation and gosec for Go; install, tests, typecheck "
            "and build for the web app; then, on main, builds one image per service, pushes to GHCR and signs each with "
            "cosign. Pin third-party actions to commit SHAs before using this in production."),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 15
    s += [H1("15. Extending toward the blueprint"),
          H2("Stage 3: more runners"),
          bullets(["Add entries to <font face='Courier'>sandbox.DefaultConfig().Images</font> for TypeScript (a Node image running with type stripping), Go and Rust.",
                   "Go and Rust compile slowly. Build images with the standard library and common HTTP crates pre-warmed, and raise the timeout for those languages only.",
                   "Add <font face='Courier'>RUNNABLE</font> entries in <font face='Courier'>web/lib/types.ts</font> so the Run button appears.",
                   "Switch the Docker runtime to gVisor before opening the runner to the public."]),
          H2("Stage 4: AI layer"),
          bullets(["Move the catalog to Postgres behind <font face='Courier'>catalog.Store</font>; add a pgvector column for embeddings of name, description and endpoint summaries.",
                   "Implement <font face='Courier'>search.Ranker</font> with a hybrid score: vector similarity plus the existing filters.",
                   "\"Ask this API\" retrieves only that API's spec and docs, then answers with citations. Grounding in the spec is what prevents invented endpoints."]),
          H2("Stage 5: growth"),
          bullets(["Drift alerts: store a hash of the live response shape per probe and flag a change against the spec.",
                   "Stack builder: let users select several APIs and generate a pipeline from the endpoint metadata that already exists.",
                   "Mobile app: the REST surface is already enough to build a read-only client."]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 16
    s += [H1("16. Limitations"),
          bullets(["Probe history is a bounded JSON file, not a time series. Restarting loses nothing, but it does not scale to long retention.",
                   "Rate limiting is per process. Run one gateway, or move the limiter to Redis before scaling out.",
                   "Doc previews use regular expressions, so unusual markup produces empty fields rather than errors.",
                   "A single probe endpoint per API means a broken secondary endpoint is not detected.",
                   "Generated Rust assumes the <font face='Courier'>reqwest</font> crate; the snippet states the Cargo line it needs.",
                   "Code generation escapes strings with JSON rules, which is correct for ASCII URLs and headers. Non-ASCII values in Rust need extra care.",
                   "Public APIs change or disappear. Probes make that visible; they do not fix it."]),
          H1("17. Pre-launch checklist"),
          table(["Check", "How"],
                [["Sandbox cannot reach the internet directly", "From inside a test container on the sandbox network: <font face='Courier'>wget -T3 https://example.com</font> must fail"],
                 ["Sandbox cannot reach metadata or LAN", "Request <font face='Courier'>169.254.169.254</font> and a LAN address through the proxy; expect denial"],
                 ["Egress allows only catalog hosts", "Run a snippet against a listed API (works) and an unlisted host (denied)"],
                 ["Timeout kills the container", "Run <font face='Courier'>while(true){}</font>; confirm <font face='Courier'>docker ps</font> is empty afterwards"],
                 ["Memory limit holds", "Allocate a large array; the container must be OOM-killed, not the host"],
                 ["Rate limits behave behind your proxy", "Confirm clients cannot spoof <font face='Courier'>X-Forwarded-For</font>"],
                 ["Runner host is isolated", "Nothing else on the VM; no credentials on disk; firewall allows only the gateway"]],
                [190, 296]),
          CondPageBreak(300)]

    # ---------------------------------------------------------------- 18
    s += [H1("18. Troubleshooting"),
          table(["Symptom", "Cause and fix"],
                [["Web shows \"Could not load the catalog\" or <font face='Courier'>/api/*</font> returns 500", "The rewrite target is baked at build time. Rebuild with <font face='Courier'>GATEWAY_URL</font> set to the gateway address."],
                 ["Gateway exits with <font face='Courier'>catalog: ...</font>", "A manifest is invalid. The message names the file and every problem. Run <font face='Courier'>make validate</font>."],
                 ["All badges say \"unverified\"", "No probe has finished yet, or <font face='Courier'>PROBES_ENABLED=false</font>, or the host has no outbound access."],
                 ["Run button returns 503", "<font face='Courier'>RUNNER_URL</font> is empty. Start the sandbox profile and set it."],
                 ["Run returns \"docker unavailable\"", "The runner cannot reach the Docker socket, or the images are not pulled. Run <font face='Courier'>docker pull node:24-alpine python:3.12-alpine</font> on the runner host."],
                 ["Sandboxed JavaScript cannot fetch", "The image must honour <font face='Courier'>NODE_USE_ENV_PROXY</font> (Node 24). Check the egress proxy log for denied CONNECT lines."],
                 ["Try it says host is not in the catalog", "Working as intended. Add the API to the catalog first."],
                 ["429 responses", "Per-IP limit reached. Raise <font face='Courier'>TRY_RPM</font> or wait the <font face='Courier'>Retry-After</font> interval."]],
                [190, 296]),
          Spacer(1, 10),
          small(f"APIScope tutorial, {version}. Generated from the working tree; code excerpts are read from the repository at build time.")]
    return s
