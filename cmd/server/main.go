package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"database_firewall/internal/config"
	"database_firewall/internal/logging"
	"database_firewall/internal/proxy"
)

var configFlag = flag.String("config", "", "to set config file path")

func main() {
	flag.Parse()

	c, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	if err := config.ValidateConfig(c); err != nil {
		log.Fatal(err)
	}

	pcfg, ccfg, rcfg := c.SplitConfig()

	connReg := proxy.NewConnectionRegister(ccfg)
	rateLimiter := proxy.NewTokenBucketLimiter(rcfg)
	admissionController := proxy.AdmissionController{
		RateLimiter: rateLimiter,
		ConnReg:     connReg,
	}

	log.Println("Starting service...")

	laddr, err := net.ResolveTCPAddr("tcp", pcfg.LocalAddress)
	if err != nil {
		log.Fatalf("Failed to resolve local address: %s", err)
	}
	raddr, err := net.ResolveTCPAddr("tcp", pcfg.RemoteAddress)
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
		ok, msg := admissionController.Admit(remoteIP)
		if !ok {
			conn.Close()
			logging.LogEvent("WARN", "connection_rejected", map[string]any{
				"client_ip": remoteIP.String(),
				"reason":    msg,
			})
		} else {
			logging.LogEvent("INFO", "connection_accepted", map[string]any{
				"client_ip":          remoteIP.String(),
				"active_connections": connReg.ActiveConnectionsCount() + 1,
			})
			p := proxy.NewProxy(pcfg, remoteIP, conn, laddr, raddr)
			go p.Start(connReg)
		}
	}

}

func handleShutdown(ln *net.TCPListener) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, os.Interrupt)

	<-sig
	log.Println("Shutting down...")
	ln.Close()
}
