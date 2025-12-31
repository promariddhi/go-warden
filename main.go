package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	c, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting service...")

	laddr, err := net.ResolveTCPAddr("tcp", c.LocalAddress)
	if err != nil {
		log.Fatalf("Failed to resolve local address: %s", err)
	}
	raddr, err := net.ResolveTCPAddr("tcp", c.RemoteAddress)
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

		p := New(conn, laddr, raddr)

		go p.start()
	}

}

func handleShutdown(ln *net.TCPListener) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)

	<-sig
	log.Println("Shutting down...")
	ln.Close()
}
