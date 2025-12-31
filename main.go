package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var cfg Config

func main() {
	c, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := validateConfig(c); err != nil {
		log.Fatal(err)
	}

	cfg = c

	connReg := NewConnectionRegister()

	log.Println("Starting service...")

	laddr, err := net.ResolveTCPAddr("tcp", cfg.LocalAddress)
	if err != nil {
		log.Fatalf("Failed to resolve local address: %s", err)
	}
	raddr, err := net.ResolveTCPAddr("tcp", cfg.RemoteAddress)
	if err != nil {
		log.Fatalf("Failed to resolve remote address: %s", err)
	}

	ln, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		log.Fatalf("Failed to open local port to listen: %s", err)
	}

	go handleShutdown(ln)

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Printf("Accept stopped: %s", err)
			break
		}
		remoteIP := net.IP(conn.RemoteAddr().(*net.TCPAddr).IP)
		p := NewProxy(remoteIP, conn, laddr, raddr)
		go p.start(connReg)
	}

}

func handleShutdown(ln *net.TCPListener) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)

	<-sig
	log.Println("Shutting down...")
	ln.Close()
}
