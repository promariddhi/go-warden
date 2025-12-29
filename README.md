# TCP Database Access Firewall

## MVP (Production-viable core)

- Generic TCP proxy (protocol-agnostic)
- Forward traffic between client and upstream database
- Max concurrent connections (global + per-IP)
- Connection rate limiting (new connections/sec per IP)
- Idle connection timeout
- Hard connection lifetime limit (optional cap)
- Graceful connection teardown and cleanup
- Structured logging (connection open/close, allow/deny)
- Metrics:
  - active connections
  - connections accepted / rejected
  - bytes in / out
- YAML-based static configuration
- Deterministic startup failure on invalid config

---

## Nice-to-haves (Post-MVP hardening)

- Hot config reload (SIGHUP)
- IP allowlist / denylist
- Per-upstream connection limits
- TLS passthrough support
- Prometheus `/metrics` endpoint
- Connection latency and duration histograms
- Admin status endpoint (read-only)
- Graceful shutdown with drain period
- Multiple upstream targets
- Basic health checks for upstream availability
