package proxy

import "net"

type AdmissionController struct {
	RateLimiter RateLimiter
	ConnReg     *ConnectionRegister
}

func (a *AdmissionController) Admit(ip net.IP) (bool, string) {
	if a.RateLimiter != nil && !a.RateLimiter.Allow(ip) {
		return false, "rate_limit"
	}

	ok, msg := a.ConnReg.TryRegister(ip)
	if !ok {
		return false, msg
	}

	return true, ""
}
