package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/goccy/go-yaml"
)

func main() {
	c, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting service...")

	if c.LocalAddress == "" {
		log.Fatal("Local Address must be defined")
	}

	if c.RemoteAddress == "" {
		log.Fatal("Remote Address must be defined")
	}

	ln, err := net.Listen("tcp", c.LocalAddress)
	if err != nil {
		log.Fatal("Server could not be started")
	}

	defer func() {
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Connection refused")
			fmt.Printf("Error trace: %s", err)
			continue
		}

		go func() {
			err := HandleConnection(conn, c.RemoteAddress)
			if err != nil {
				fmt.Println("Request denied")
				fmt.Printf("Error trace: %s", err)
			}
		}()
	}
}

func HandleConnection(local net.Conn, remoteAddress string) error {
	fmt.Printf("forwarding request to %s\n", remoteAddress)
	remote, err := net.Dial("tcp", remoteAddress)
	if err != nil {
		return err
	}

	go func() error {
		return Copier(local, remote)
	}()

	go func() error {
		return Copier(remote, local)
	}()

	return nil
}

func Copier(w io.Writer, r io.Reader) error {
	_, err := io.Copy(w, r)
	return err
}

func LoadConfig() (Config, error) {
	config, err := os.Open("config.yml")
	if err != nil {
		return Config{}, err
	}

	defer func() {
		_ = config.Close()
	}()

	config_b, err := io.ReadAll(config)
	if err != nil {
		return Config{}, err
	}

	c := Config{}

	if err := yaml.Unmarshal(config_b, &c); err != nil {
		return Config{}, err
	}

	return c, nil

}

type Config struct {
	LocalAddress  string `yaml:"local_address"`
	RemoteAddress string `yaml:"remote_address"`
}
