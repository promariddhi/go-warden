# Database Firewall 

## Implemented

- Generic TCP listener (protocol-agnostic)
- Accept loop with one proxy instance per connection
- Bidirectional byte-for-byte forwarding (client ↔ upstream)
- Coordinated teardown on first read/write failure
- Graceful shutdown on `SIGINT` / `SIGTERM`
- Static configuration via YAML

## Next

- Active connection tracking
- Global / per-IP connection limits
- Idle connection timeouts
- Structured connection lifecycle logging
- In-memory metrics (connections, bytes in/out)

## Blogs
[Part 1](https://medium.com/@promariddhi/building-a-database-firewall-part-1-tcp-proxy-4134026ef739)

