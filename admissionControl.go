package main

import "net"

type AdmissionController struct {
	rateLimiter RateLimiter
	connReg     *ConnectionRegister
}

func (a *AdmissionController) Admit(ip net.IP) (bool, string) {
	if a.rateLimiter != nil && !a.rateLimiter.Allow(ip) {
		return false, "rate_limit"
	}

	ok, msg := a.connReg.TryRegister(ip)
	if !ok {
		return false, msg
	}

	return true, ""
}
