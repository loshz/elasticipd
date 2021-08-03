package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// startHTTP creates and starts a local webserver used to expose a health
// check and prometheus metrics
func startHTTP(port int) {
	// configure http server with 10s timeoutes
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// configure a health check handler
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// register and configure a prometheus metrics handler
	prometheus.MustRegister(criticalErrors)
	http.Handle("/metrics", promhttp.Handler())

	// we don't care about errors from the server as the caller of the health check
	// should interpret failed responses as fatal
	log.Info().Msgf("started local http server on :%d", port)
	srv.ListenAndServe()
}
