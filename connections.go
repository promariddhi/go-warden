package main

import (
	"net"
	"sync"
)

type ConnectionRegister struct {
	mu                sync.Mutex
	ActiveConnections int64
	ConnectionsByIP   map[string]int64

	//--------metrics----------
	ConnectionsAccepted, ConnectionsRejected int64
}

func NewConnectionRegister() *ConnectionRegister {
	return &ConnectionRegister{
		ConnectionsByIP: make(map[string]int64),
	}
}

func (r *ConnectionRegister) Register(ip net.IP) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.canAccept(ip) {
		r.ConnectionsRejected += 1
		return false
	}
	r.ActiveConnections += 1
	r.ConnectionsAccepted += 1
	ct, ok := r.ConnectionsByIP[ip.String()]
	if !ok {
		ct = 0
	}
	r.ConnectionsByIP[ip.String()] = ct + 1
	return true
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

func (r *ConnectionRegister) canAccept(ip net.IP) bool {

	if r.ActiveConnections < cfg.ConnectionLimit && r.ConnectionsByIP[ip.String()] < cfg.PerIPConnectionLimit {
		return true
	}
	return false
}
