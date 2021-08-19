package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

var version = "dev"

func main() {
	eIP := flag.String("elastic-ip", "", "Elastic IP address to associate")
	interval := flag.String("interval", "30s", "Attempt association every interval")
	port := flag.Int("port", 8081, "Local HTTP server port")
	reassoc := flag.Bool("reassoc", true, "Allow Elastic IP to be reassociated without failure")
	region := flag.String("region", "", "AWS region hosting the Elastic IP and EC2 instance")
	retries := flag.Int("retries", 3, "Maximum number of association retries before fatally exiting")
	flag.Parse()

	// check required flags are set
	if *eIP == "" || *region == "" {
		flag.Usage()
		os.Exit(1)
	}

	// configure global logger defaults
	log.Logger = log.Logger.With().Fields(map[string]interface{}{
		"service": "elasticipd",
		"version": version,
	}).Logger()

	// check Elastic IP is valid ipv4
	if net.ParseIP(*eIP).To4() == nil {
		log.Fatal().Msgf("invalid ipv4: %q", *eIP)
	}

	// parse poll interval
	pollInt, err := time.ParseDuration(*interval)
	if err != nil {
		log.Fatal().Msgf("invalid poll interval: %q", *interval)
	}

	// configure a channel to listen for exit signals in order to perform
	// a graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// start the local http server
	go startHTTP(*port)

	// configure aws services
	svc, err := newSvc(*region)
	if err != nil {
		log.Fatal().Err(err).Msg("error configuring aws services")
	}

	// start the main thread
	log.Info().Msg("service started")
	poll(stop, pollInt, svc, *eIP, *reassoc, *retries)
}

func poll(stop chan os.Signal, interval time.Duration, s svc, eIP string, reassoc bool, maxRetries int) {
	var retries int

	// start a ticker at given intervals
	t := time.NewTicker(interval)
	for {
		if retries == maxRetries {
			log.Fatal().Msg("maximum amount of retries reached, exiting")
		}

		select {
		case <-stop:
			log.Info().Msg("received stop signal, attempting graceful shutdown")

			// attempt to disassociate the address before shutting down
			assoc, err := s.getAssociation(eIP)
			if err != nil {
				log.Fatal().Err(err).Msg("error describing elastic ip")
			}
			if err := s.disassociateAddr(assoc); err != nil {
				log.Fatal().Err(err).Msg("error disassociating elastic ip")
			}

			log.Info().Str("instance_id", assoc.instanceID).Msg("elastic ip disassociated from current instance")
			os.Exit(0)
		case <-t.C:
			// get elastic ip association and current instance details
			assoc, err := s.getAssociation(eIP)
			if err != nil {
				log.Error().Err(err).Msg("error describing elastic ip")
				retries++
				continue
			}
			ins, err := s.getInstanceDetails()
			if err != nil {
				log.Error().Err(err).Msg("error getting instance details")
				retries++
				continue
			}

			// if the Elastic IP is already associated to the current EC2 instance, skip
			if assoc.instanceID == ins.id {
				continue
			}

			// attempt to associate elastic ip
			if err := s.associateAddr(assoc, ins, reassoc); err != nil {
				log.Error().Err(err).Msg("error associating elastic ip")
				retries++
				continue
			}

			retries = 0
			log.Info().Str("instance_id", assoc.instanceID).Msg("elastic ip associated to current instance")
		}
	}
}
