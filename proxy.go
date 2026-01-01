package main

import (
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Proxy struct {
	ip                net.IP
	laddr, raddr      *net.TCPAddr
	lconn, rconn      *net.TCPConn
	startTime         time.Time
	inBytes, outBytes int

	//------error handling--------
	errOnce sync.Once
	errsig  chan struct{}
}

func NewProxy(ip net.IP, lconn *net.TCPConn, laddr, raddr *net.TCPAddr) *Proxy {
	return &Proxy{
		ip:        ip,
		lconn:     lconn,
		laddr:     laddr,
		raddr:     raddr,
		startTime: time.Now(),
		errsig:    make(chan struct{}),
	}
}

func (p *Proxy) start(r *ConnectionRegister) {
	defer p.lconn.Close()
	log.Printf("Connecting to %s...", p.raddr)

	//--------------registration logic-----------------
	r.Register(p.ip)
	defer r.Unregister(p.ip)

	//----------setting  idle timeout------------------
	if cfg.IdleTimeoutSeconds > 0 {
		deadline := time.Now().Add(time.Duration(cfg.IdleTimeoutSeconds) * time.Second)
		p.lconn.SetDeadline(deadline)
		p.rconn.SetDeadline(deadline)
	}

	//--------------Dial and copy------------------------
	var err error
	p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
	if err != nil {
		log.Printf("Remote connection failed: %s", err)
		return
	}

	defer p.rconn.Close()

	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)

	<-p.errsig
	logEvent("INFO", "connection_closed", map[string]any{
		"client_ip":   p.ip.String(),
		"duration_ms": time.Since(p.startTime),
		"bytes_in":    p.inBytes,
		"bytes_out":   p.outBytes,
	})

}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed: %s\n", err)
			return
		}
		p.inBytes += n
		p.refreshDeadline()
		b := buff[:n]
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed: %s\n", err)
			return
		}
		p.outBytes += n
		p.refreshDeadline()
	}
}

func (p *Proxy) err(s string, err error) {
	p.errOnce.Do(func() {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			logEvent("INFO", "connection_closed", map[string]any{
				"client_ip": p.ip.String(),
				"reason":    "idle_timeout",
			})
		}
		if err != io.EOF {
			log.Printf("stage=%s error=%s", s, err)
		}
		close(p.errsig)
	})
}

func (p *Proxy) refreshDeadline() {
	if cfg.IdleTimeoutSeconds <= 0 {
		return
	}
	deadline := time.Now().Add(time.Duration(cfg.IdleTimeoutSeconds) * time.Second)
	p.lconn.SetDeadline(deadline)
	p.rconn.SetDeadline(deadline)

}
