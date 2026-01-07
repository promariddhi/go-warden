package proxy

import (
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"database_firewall/internal/config"
	"database_firewall/internal/logging"
)

type Proxy struct {
	cfg               config.ProxyConfig
	ip                net.IP
	laddr, raddr      *net.TCPAddr
	lconn, rconn      *net.TCPConn
	startTime         time.Time
	inBytes, outBytes int64

	//------error handling--------
	errOnce sync.Once
	errsig  chan struct{}
}

func NewProxy(cfg *config.ProxyConfig, ip net.IP, lconn *net.TCPConn, laddr, raddr *net.TCPAddr) *Proxy {
	return &Proxy{
		cfg:       *cfg,
		ip:        ip,
		lconn:     lconn,
		laddr:     laddr,
		raddr:     raddr,
		startTime: time.Now(),
		errsig:    make(chan struct{}),
	}
}

func (p *Proxy) Start(r *ConnectionRegister) {
	defer p.lconn.Close()
	log.Printf("Connecting to %s...", p.raddr)

	//--------------registration logic-----------------
	defer r.Unregister(p.ip)

	//--------------Dial and copy------------------------
	var err error
	p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
	if err != nil {
		log.Printf("Remote connection failed: %s", err)
		return
	}

	defer p.rconn.Close()

	//----------setting  idle timeout------------------
	if p.cfg.IdleTimeoutSeconds > 0 {
		deadline := time.Now().Add(time.Duration(p.cfg.IdleTimeoutSeconds) * time.Second)
		p.lconn.SetDeadline(deadline)
		p.rconn.SetDeadline(deadline)
	}

	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)

	<-p.errsig
	logging.LogEvent("INFO", "connection_closed", map[string]any{
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
		atomic.AddInt64(&p.inBytes, int64(n))
		p.refreshDeadline()
		b := buff[:n]
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed: %s\n", err)
			return
		}
		atomic.AddInt64(&p.outBytes, int64(n))
		p.refreshDeadline()
	}
}

func (p *Proxy) err(s string, err error) {
	p.errOnce.Do(func() {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			logging.LogEvent("INFO", "connection_closed", map[string]any{
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
	if p.cfg.IdleTimeoutSeconds <= 0 {
		return
	}
	deadline := time.Now().Add(time.Duration(p.cfg.IdleTimeoutSeconds) * time.Second)
	p.lconn.SetDeadline(deadline)
	p.rconn.SetDeadline(deadline)

}
