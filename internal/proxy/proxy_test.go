package proxy

import (
	"net"
	"sync"
	"testing"
	"time"

	"database_firewall/internal/config"
)

/*
-------------------------------------------------
Helpers
-------------------------------------------------
*/

// startUpstream starts a TCP listener that accepts connections
// and optionally closes them after a delay.
func startUpstream(t *testing.T, closeDelay time.Duration) (*net.TCPAddr, func()) {
	t.Helper()

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				return
			}
			go func(c *net.TCPConn) {
				if closeDelay > 0 {
					time.Sleep(closeDelay)
				}
				c.Close()
			}(conn)
		}
	}()

	cleanup := func() {
		ln.Close()
		<-done
	}

	return ln.Addr().(*net.TCPAddr), cleanup
}

func dialClient(t *testing.T, addr *net.TCPAddr) *net.TCPConn {
	t.Helper()

	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func testProxyConfig(idleSecs int64) *config.ProxyConfig {
	return &config.ProxyConfig{
		LocalAddress:       "127.0.0.1:0",
		RemoteAddress:      "127.0.0.1:0",
		IdleTimeoutSeconds: idleSecs,
	}
}

/*
-------------------------------------------------
Test: immediate client close → unregister happens
-------------------------------------------------
*/
func TestProxy_NoLeakOnImmediateClientClose(t *testing.T) {
	ccfg := &config.ConnectionConfig{
		ConnectionLimit:      1,
		PerIPConnectionLimit: 1,
	}
	reg := NewConnectionRegister(ccfg)

	ip := net.ParseIP("10.0.0.1")

	upAddr, cleanup := startUpstream(t, 0)
	defer cleanup()

	// client side TCP connection (this is lconn)
	lconn := dialClient(t, upAddr)
	defer lconn.Close()

	ac := AdmissionController{
		RateLimiter: nil,
		ConnReg:     reg,
	}

	ok, _ := ac.Admit(ip)
	if !ok {
		return
	}

	p := NewProxy(
		testProxyConfig(0),
		ip,
		lconn,
		nil,
		upAddr,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Start(reg)
	}()

	// immediate client close
	lconn.Close()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("proxy did not exit")
	}

	if reg.ActiveConnectionsCount() != 0 {
		t.Fatalf("connection leak: active=%d", reg.ActiveConnectionsCount())
	}
}

/*
-------------------------------------------------
Test: upstream dial failure → unregister happens
-------------------------------------------------
*/
func TestProxy_NoLeakOnUpstreamDialFailure(t *testing.T) {
	ccfg := &config.ConnectionConfig{
		ConnectionLimit:      1,
		PerIPConnectionLimit: 1,
	}
	reg := NewConnectionRegister(ccfg)

	ip := net.ParseIP("10.0.0.1")

	// client listener (acts as accepted socket)
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	clientDone := make(chan *net.TCPConn)
	go func() {
		conn, _ := ln.AcceptTCP()
		clientDone <- conn
	}()

	lconn := dialClient(t, ln.Addr().(*net.TCPAddr))
	serverConn := <-clientDone
	defer lconn.Close()
	defer serverConn.Close()

	// unreachable upstream
	badAddr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 1,
	}

	ac := AdmissionController{
		RateLimiter: nil,
		ConnReg:     reg,
	}

	ok, _ := ac.Admit(ip)
	if !ok {
		return
	}

	p := NewProxy(
		testProxyConfig(0),
		ip,
		serverConn,
		nil,
		badAddr,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Start(reg)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("proxy did not exit on upstream dial failure")
	}

	if reg.ActiveConnectionsCount() != 0 {
		t.Fatalf("leak after upstream failure: active=%d", reg.ActiveConnectionsCount())
	}
}

/*
-------------------------------------------------
Test: idle timeout closes connection + unregisters
-------------------------------------------------
*/
func TestProxy_IdleTimeoutTriggersUnregister(t *testing.T) {
	ccfg := &config.ConnectionConfig{
		ConnectionLimit:      1,
		PerIPConnectionLimit: 1,
	}
	reg := NewConnectionRegister(ccfg)

	ip := net.ParseIP("10.0.0.1")

	upAddr, cleanup := startUpstream(t, 2*time.Second)
	defer cleanup()

	lconn := dialClient(t, upAddr)
	defer lconn.Close()

	ac := AdmissionController{
		RateLimiter: nil,
		ConnReg:     reg,
	}

	ok, _ := ac.Admit(ip)
	if !ok {
		return
	}

	p := NewProxy(
		testProxyConfig(1), // idle timeout = 1s
		ip,
		lconn,
		nil,
		upAddr,
	)

	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Start(reg)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy did not exit after idle timeout")
	}

	if reg.ActiveConnectionsCount() != 0 {
		t.Fatalf("leak after idle timeout: active=%d", reg.ActiveConnectionsCount())
	}
}

/*
-------------------------------------------------
Test: many proxies concurrently → no leaks
-------------------------------------------------
*/
func TestProxy_NoLeaksUnderConcurrency(t *testing.T) {
	ccfg := &config.ConnectionConfig{
		ConnectionLimit:      20,
		PerIPConnectionLimit: 20,
	}
	reg := NewConnectionRegister(ccfg)

	ip := net.ParseIP("10.0.0.1")

	upAddr, cleanup := startUpstream(t, 10*time.Millisecond)
	defer cleanup()

	const n = 10
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// client side connection
			lconn := dialClient(t, upAddr)
			defer lconn.Close()

			ac := AdmissionController{
				RateLimiter: nil,
				ConnReg:     reg,
			}

			ok, _ := ac.Admit(ip)
			if !ok {
				return
			}

			p := NewProxy(
				testProxyConfig(0),
				ip,
				lconn,
				nil,
				upAddr,
			)

			done := make(chan struct{})
			go func() {
				defer close(done)
				p.Start(reg)
			}()

			lconn.Close()
			<-done
		}()
	}

	wg.Wait()

	if reg.ActiveConnectionsCount() != 0 {
		t.Fatalf("leak after concurrency: active=%d", reg.ActiveConnectionsCount())
	}
}
