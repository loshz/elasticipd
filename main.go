package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envElasticIP = "ELASTIC_IP"
	envAWSRegion = "AWS_REGION"
)

func main() {
	// check for valid Elastic IP
	ip := net.ParseIP(os.Getenv(envElasticIP))
	if ip.To4() == nil {
		log.Fatalf("invalid ipv4: %s", ip)
	}

	// check for valid AWS region
	region := os.Getenv(envAWSRegion)
	if region == "" {
		log.Fatalf("missing aws region: %s", envAWSRegion)
	}

	// configure a channel to listen for exit signals in order to perform
	// a graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	// start a ticker at 10s intervals
	t := time.NewTicker(10 * time.Second)

	// configure and run web server for health check,
	// we don't care about any errors as the healthcheck caller
	// should interpret this as fatal
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	go http.ListenAndServe(":8081", nil)

	log.Printf("service started, will attempt to allocate Elastic IP: %s to current instance every 10s", ip)
	log.Printf("health check registered on localhost:8081/healthz")

	for {
		select {
		case <-stop:
			log.Println("received stop signal, attempting graceful shutdown")

			// stop ticker
			t.Stop()

			// passing shutdown = true will ensure the Elastic IP is disassociated only
			if err := assignElasticIP(region, ip.String(), true); err != nil {
				log.Fatalf("shutdown error: %v", err)
			}

			os.Exit(0)
		case <-t.C:
			// passing shutdown = false will ensure the Elastic IP is disassociated from any
			// current associations, and associated to the current instance
			if err := assignElasticIP(region, ip.String(), false); err != nil {
				log.Printf("assigning Elastic IP: %v", err)
			}
		}
	}
}
