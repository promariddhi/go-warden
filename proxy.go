package main

import (
	"io"
	"log"
	"net"
	"sync"
)

type Proxy struct {
	laddr, raddr *net.TCPAddr
	lconn, rconn io.ReadWriteCloser
	errOnce      sync.Once
	errsig       chan struct{}
}

func New(lconn *net.TCPConn, laddr, raddr *net.TCPAddr) *Proxy {
	return &Proxy{
		lconn:  lconn,
		laddr:  laddr,
		raddr:  raddr,
		errsig: make(chan struct{}),
	}
}

func (p *Proxy) start() {
	defer p.lconn.Close()
	log.Printf("Connecting to %s...", p.raddr)
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
	log.Printf("Closed!")
}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed: %s\n", err)
			return
		}
		b := buff[:n]
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed: %s\n", err)
			return
		}
	}
}

func (p *Proxy) err(s string, err error) {
	p.errOnce.Do(func() {
		if err != io.EOF {
			log.Printf(s, err)
		}
		close(p.errsig)
	})
}
