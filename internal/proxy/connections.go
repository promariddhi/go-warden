package proxy

import (
	"net"
	"sync"

	"database_firewall/internal/config"
)

type ConnectionRegister struct {
	mu                sync.Mutex
	cfg               config.ConnectionConfig
	ActiveConnections int64
	ConnectionsByIP   map[string]int64

	//--------metrics----------
	ConnectionsAccepted, ConnectionsRejected int64
}

func NewConnectionRegister(cfg *(config.ConnectionConfig)) *ConnectionRegister {
	return &ConnectionRegister{
		cfg:             *cfg,
		ConnectionsByIP: make(map[string]int64),
	}
}

func (r *ConnectionRegister) TryRegister(ip net.IP) (bool, string) {
	key := ip.String()
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ActiveConnections >= r.cfg.ConnectionLimit {
		r.ConnectionsRejected += 1
		return false, "connection_limit"
	}

	if r.ConnectionsByIP[key] >= r.cfg.PerIPConnectionLimit {
		r.ConnectionsRejected += 1
		return false, "per_ip_limit"
	}

	r.ConnectionsAccepted += 1
	r.register(ip)
	return true, ""
}

func (r *ConnectionRegister) register(ip net.IP) {
	r.ActiveConnections += 1
	ct, ok := r.ConnectionsByIP[ip.String()]
	if !ok {
		ct = 0
	}
	r.ConnectionsByIP[ip.String()] = ct + 1
}

func (r *ConnectionRegister) Unregister(ip net.IP) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ActiveConnections -= 1
	ct, ok := r.ConnectionsByIP[ip.String()]
	if !ok {
		return
	}
	if ct <= 1 {
		delete(r.ConnectionsByIP, ip.String())
		return
	}
	r.ConnectionsByIP[ip.String()] = ct - 1
}

func (r *ConnectionRegister) ActiveConnectionsCount() int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ActiveConnections
}

func (r *ConnectionRegister) IPConnectionsCount(ip net.IP) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.ConnectionsByIP[ip.String()]
}
