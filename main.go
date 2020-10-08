package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envElasticIP = "ELASTIC_IP"
)

func main() {
	ip := net.ParseIP(os.Getenv(envElasticIP))
	if ip.To4() == nil {
		log.Fatalf("invalid ipv4: %s", ip)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	t := time.NewTicker(60 * time.Second)

	for {
		select {
		case <-quit:
			log.Println("received stop signal, attempting graceful shut down")

			// stop ticker
			t.Stop()

			if err := assignElasticIP(ip.String(), true); err != nil {
				log.Fatalf("shutdown error: %v", err)
			}

			log.Printf("successfully disassociated address: %s", ip)
			os.Exit(0)
		case <-t.C:
			if err := assignElasticIP(ip.String(), false); err != nil {
				log.Printf("assigning Elastic IP: %v", err)
			}
		}
	}
}
