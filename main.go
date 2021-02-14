package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

const (
	envElasticIP    = "ELASTIC_IP"
	envAWSRegion    = "AWS_REGION"
	envPollInterval = "POLL_INTERVAL"
	envPort         = "PORT"
)

var (
	// Prometheus gauge for storing number of failed ip associations
	criticalError = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "elasticipd_critical_error",
		Help: "Set to 0 if OK or 1 if there is an error associating/disassociating the Elastic IP",
	})
)

func main() {
	// check for valid Elastic IP
	ip := net.ParseIP(os.Getenv(envElasticIP))
	if ip.To4() == nil {
		log.Fatal().Msgf("invalid ipv4: %s", ip)
	}

	// check for valid AWS region
	region := os.Getenv(envAWSRegion)
	if region == "" {
		log.Fatal().Msgf("missing aws region: %s", envAWSRegion)
	}

	// parse poll interval
	poll, err := time.ParseDuration(os.Getenv(envPollInterval))
	if err != nil {
		log.Fatal().Msgf("invalid poll interval: %s", poll)
	}

	// parse port
	port := 8081
	if p := os.Getenv(envPort); p != "" {
		port, err = strconv.Atoi(p)
		if err != nil {
			log.Fatal().Msgf("invalid port: %s: %v", p, err)
		}
	}

	// configure a channel to listen for exit signals in order to perform
	// a graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// start the local http server
	srv := startHTTP(port)

	// start a ticker at given intervals
	t := time.NewTicker(poll)
	log.Info().Msgf("service started, will attempt to allocate Elastic IP %s to current instance every %s", ip, poll)

	for {
		select {
		case <-stop:
			log.Info().Msg("received stop signal, attempting graceful shutdown")

			// stop ticker
			t.Stop()

			// passing shutdown = true will ensure the Elastic IP is disassociated only
			if err := assignElasticIP(region, ip.String(), true); err != nil {
				log.Fatal().Err(err).Msg("error shutting down gracefully")
			}

			// gracefully shutdown local http server
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := srv.Shutdown(ctx); err != nil {
				cancel()
				log.Fatal().Err(err).Msg("error shuting down http server")
			}

			os.Exit(0)
		case <-t.C:
			// passing shutdown = false will ensure the Elastic IP is disassociated from any
			// current associations, and associated to the current instance
			if err := assignElasticIP(region, ip.String(), false); err != nil {
				criticalError.Set(1)
				log.Error().Err(err).Msg("assigning Elastic IP")
			}

			// reset the error counter on success
			criticalError.Set(0)
		}
	}
}
