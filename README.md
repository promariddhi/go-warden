# TCP Database Access Firewall

## TODO

- [x] TCP proxy (protocol-agnostic)
- [ ] Forward traffic between client and upstream database
- [ ] Max concurrent connections (global + per-IP)
- [ ] Connection rate limiting (new connections/sec per IP)
- [ ] Idle connection timeout
- [ ] Hard connection lifetime limit (optional cap)
- [ ] Graceful connection teardown and cleanup
- [ ] Structured logging (connection open/close, allow/deny)
- Metrics:
  - [ ] active connections
  - [ ] connections accepted / rejected
  - [ ]bytes in / out
- [ ] YAML-based static configuration
- [ ] Deterministic startup failure on invalid config
