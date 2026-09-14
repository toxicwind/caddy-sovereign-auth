# caddy-sovereign-auth

Honest description up front: **this repo is two unrelated halves** that share a
name and nothing else.

1. **A Go Caddy v2 HTTP auth middleware** (`http.handlers.sovereign_auth`) —
   API-key / Tailscale-identity gate for Caddy frontends.
2. **A JavaScript "agentic lens" perception harness** (`lib/lens-orchestrator.js`
   + four lenses under `src/_11ty/lenses/`) with its own CI workflows.

They were committed into the same repo; they are not one product.

---

## Half 1 — Caddy auth middleware

`sovereignauth.go` registers a Caddy HTTP handler module:

- **Module ID:** `http.handlers.sovereign_auth`
- **Go module:** `github.com/toxicwind/caddy-sovereign-auth` (go 1.26.5, Caddy
  v2.11.4; see `go.mod` — all deps are currently `// indirect`, so run
  `go mod tidy` before building)

### Config

```json
{
  "handler": "sovereign_auth",
  "api_keys": ["key-one", "key-two"],
  "tailscale_auth": true,
  "trusted_networks": ["10.0.0."]
}
```

| Field | Type | Behavior |
|---|---|---|
| `api_keys` | `[]string` | Accepts `Authorization: Bearer <key>` or the `X-API-Key` header. If the list is empty, this check is skipped. |
| `tailscale_auth` | `bool` | Accepts the request when the `X-Tailscale-User-Login` header is non-empty (i.e. Caddy is behind a Tailscale-served listener that sets it). |
| `trusted_networks` | `[]string` | Prefix bypass — see warning below. |

### Auth order

1. Trusted-network prefix match → allow
2. Tailscale header present (when `tailscale_auth: true`) → allow
3. API key matches → allow
4. Otherwise → `401` with body `{"error":"unauthorized"}` and header
   `WWW-Authenticate: Bearer realm="sovereign"`

### ⚠️ Trusted-network matching is NOT CIDR

`trusted_networks` entries are matched with a crude
`strings.HasPrefix(clientIP, strings.Split(cidr, "/")[0])` against
`r.RemoteAddr` (which includes the port). There is no CIDR parsing — an entry
like `"10.0.0."` matches any remote address starting with that string. Treat
this as a rough allowlist convenience, not a security boundary.

### Build

No Caddyfile, Dockerfile, or build script ships in this repo. The standard
route for a Caddy module is:

```bash
go mod tidy
xcaddy build --with github.com/toxicwind/caddy-sovereign-auth
```

---

## Half 2 — Agentic lens harness

A small Node.js "perception layer": drop a `lens_*.js` module into
`src/_11ty/lenses/` exporting `name` and an `analyze(content, meta)` function,
and `LensOrchestrator` picks it up automatically.

```js
const { LensOrchestrator } = require('./lib/lens-orchestrator');
const orch = new LensOrchestrator();           // lensDir defaults to src/_11ty/lenses (absolute)
await orch.discover();                        // logs "Discovered N lens profiles"
const all = await orch.analyzeAll(content);   // { cryptographic: {...}, osint: {...}, ... }
```

### Shipped lenses

| Lens | `name` | Does |
|---|---|---|
| `lens_cryptographic.js` | `cryptographic` | Pattern detection for keys, hashes, tokens (entropy flagging) |
| `lens_osint.js` | `osint` | Infrastructure reconnaissance / digital forensics (URL extraction) |
| `lens_stylometric.js` | `stylometric` | Linguistic fingerprinting / authorship attribution (confidence score) |
| `lens_tectonic.js` | `tectonic` | Repository health and drift detection |

### Configuration

Copy `.env.example` to `.env`. Never hardcode secrets — inject via env or a
runtime vault.

| Variable | Default | Purpose |
|---|---|---|
| `LENS_ENABLED` | `true` | Master switch for the lens layer |
| `LENS_SWARM_CONCURRENCY` | `4` | Lens parallelism |
| `LENS_AUTO_COMMIT` | `false` | Whether lens runs may commit |
| `SWARM_MAX_CONCURRENCY` | `4` | Swarm-level parallelism |
| `SWARM_TIMEOUT_MS` | `30000` | Per-task timeout |
| `GITHUB_TOKEN` | *(empty)* | GitHub API access |
| `OPHEL_VAULT_PATH` | *(empty)* | Vault path for secrets |
| `MODELBEATS_API_ENDPOINT` | *(empty)* | Modelbeats API |
| `ZEDRA_DAEMON_HOST` | `localhost:17357` | Zedra daemon |
| `MUSEPOOL_CDN_PRIMARY` | `alibaba` | CDN selection |
| `MCP_TRANSPORT` | `stdio` | MCP transport |

### CI

- **`agentic-lens-ci.yml`** — runs on push to `main`, PRs, and manual dispatch.
  Discovers lenses and smoke-tests the stylometric, cryptographic, and OSINT
  lenses against canned inputs, then runs an MCP smoke job (skipped — no
  `services/mcp-stack` in this repo). Bun install is skipped because there is
  no `package.json`.
- **`tectonic-drift.yml`** — daily `06:00 UTC` cron (plus manual dispatch)
  running the tectonic lens for drift detection.

### History

The lens system and both workflows landed 2026-09-03/04; CI repairs followed
on 2026-09-14 (unterminated-quote fix, dropped a broken bun/npm install step).

---

## Also in this repo

- **`AGENTS.md`** — hard rules for AI agents working in this repo (read it
  before touching anything).
- **`go.sum`** — Go dependency checksums.

## License

No LICENSE file is present in the tree. Treat the code as all-rights-reserved
until the owner adds one.
