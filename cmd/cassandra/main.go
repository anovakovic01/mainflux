package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers"
	"github.com/mainflux/mainflux/writers/cassandra"
	nats "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	sep = ","

	defNatsURL  = nats.DefaultURL
	defPort     = "8180"
	defCluster  = "127.0.0.1"
	defKeyspace = "mainflux"

	envNatsURL  = "MF_NATS_URL"
	envPort     = "MF_CASSANDRA_WRITER_PORT"
	envCluster  = "MF_CASSANDRA_WRITER_DB_CLUSTER"
	envKeyspace = "MF_CASSANDRA_WRITER_DB_KEYSPACE"
)

type config struct {
	natsURL  string
	port     string
	cluster  string
	keyspace string
}

func main() {
	cfg := loadConfig()

	logger := log.New(os.Stdout)

	nc, err := nats.Connect(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	session, err := cassandra.Connect(strings.Split(cfg.cluster, sep), cfg.keyspace)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}
	defer session.Close()

	if err := cassandra.Initialize(session); err != nil {
		logger.Error(fmt.Sprintf("Failed to initialize keyspace in Cassandra cluster: %s", err))
		os.Exit(1)
	}

	repo := cassandra.New(session)
	repo = writers.LoggingMiddleware(repo, logger)
	repo = writers.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_writer",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_writer",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	if err := writers.Start(nc, logger, repo); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Cassandra writer: %s", err))
	}

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go func() {
		p := fmt.Sprintf(":%s", cfg.port)
		logger.Info(fmt.Sprintf("Cassandra writer service started, exposed port %s", cfg.port))
		errs <- http.ListenAndServe(p, cassandra.MakeHandler())
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Cassandra writer service terminated: %s", err))
}

func loadConfig() config {
	return config{
		natsURL:  mainflux.Env(envNatsURL, defNatsURL),
		port:     mainflux.Env(envPort, defPort),
		cluster:  mainflux.Env(envCluster, defCluster),
		keyspace: mainflux.Env(envKeyspace, defKeyspace),
	}
}
