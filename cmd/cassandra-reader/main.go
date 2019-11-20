// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
	authzgrpc "github.com/mainflux/mainflux/authz/api/grpc"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/cassandra"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	sep = ","

	defLogLevel      = "error"
	defPort          = "8180"
	defCluster       = "127.0.0.1"
	defKeyspace      = "mainflux"
	defDBUsername    = ""
	defDBPassword    = ""
	defDBPort        = "9042"
	defThingsURL     = "localhost:8181"
	defClientTLS     = "false"
	defCACerts       = ""
	defJaegerURL     = ""
	defThingsTimeout = "1" // in seconds
	defAuthzURL      = "localhost:8181"

	envLogLevel      = "MF_CASSANDRA_READER_LOG_LEVEL"
	envPort          = "MF_CASSANDRA_READER_PORT"
	envCluster       = "MF_CASSANDRA_READER_DB_CLUSTER"
	envKeyspace      = "MF_CASSANDRA_READER_DB_KEYSPACE"
	envDBUsername    = "MF_CASSANDRA_READER_DB_USERNAME"
	envDBPassword    = "MF_CASSANDRA_READER_DB_PASSWORD"
	envDBPort        = "MF_CASSANDRA_READER_DB_PORT"
	envThingsURL     = "MF_THINGS_URL"
	envClientTLS     = "MF_CASSANDRA_READER_CLIENT_TLS"
	envCACerts       = "MF_CASSANDRA_READER_CA_CERTS"
	envJaegerURL     = "MF_JAEGER_URL"
	envThingsTimeout = "MF_CASSANDRA_READER_THINGS_TIMEOUT"
	envAuthzURL      = "MF_AUTHZ_URL"
)

type config struct {
	logLevel      string
	port          string
	dbCfg         cassandra.DBConfig
	thingsURL     string
	clientTLS     bool
	caCerts       string
	jaegerURL     string
	thingsTimeout time.Duration
	authzURL      string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	session := connectToCassandra(cfg.dbCfg, logger)
	defer session.Close()

	thingsConn := connectToGRPC(cfg, cfg.thingsURL, logger)
	defer thingsConn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	authzConn := connectToGRPC(cfg, cfg.authzURL, logger)
	defer authzConn.Close()

	authzTracer, authzCloser := initJaeger("authz", cfg.jaegerURL, logger)
	defer authzCloser.Close()

	authz := authzgrpc.NewClient(authzConn, authzTracer)

	authn := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsTimeout)
	repo := newService(session, logger)

	errs := make(chan error, 2)

	go startHTTPServer(repo, authz, authn, cfg.port, errs, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Cassandra reader service terminated: %s", err))
}

func loadConfig() config {
	dbPort, err := strconv.Atoi(mainflux.Env(envDBPort, defDBPort))
	if err != nil {
		log.Fatal(err)
	}

	dbCfg := cassandra.DBConfig{
		Hosts:    strings.Split(mainflux.Env(envCluster, defCluster), sep),
		Keyspace: mainflux.Env(envKeyspace, defKeyspace),
		Username: mainflux.Env(envDBUsername, defDBUsername),
		Password: mainflux.Env(envDBPassword, defDBPassword),
		Port:     dbPort,
	}

	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	timeout, err := strconv.ParseInt(mainflux.Env(envThingsTimeout, defThingsTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	return config{
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		port:          mainflux.Env(envPort, defPort),
		dbCfg:         dbCfg,
		thingsURL:     mainflux.Env(envThingsURL, defThingsURL),
		clientTLS:     tls,
		caCerts:       mainflux.Env(envCACerts, defCACerts),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		thingsTimeout: time.Duration(timeout) * time.Second,
		authzURL:      mainflux.Env(envAuthzURL, defAuthzURL),
	}
}

func connectToCassandra(dbCfg cassandra.DBConfig, logger logger.Logger) *gocql.Session {
	session, err := cassandra.Connect(dbCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
}

func connectToGRPC(cfg config, url string, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(url, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to %s service: %s", url, err))
		os.Exit(1)
	}
	return conn
}

func initJaeger(svcName, url string, logger logger.Logger) (opentracing.Tracer, io.Closer) {
	if url == "" {
		return opentracing.NoopTracer{}, ioutil.NopCloser(nil)
	}

	tracer, closer, err := jconfig.Configuration{
		ServiceName: svcName,
		Sampler: &jconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jconfig.ReporterConfig{
			LocalAgentHostPort: url,
			LogSpans:           true,
		},
	}.NewTracer()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger client: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func newService(session *gocql.Session, logger logger.Logger) readers.MessageRepository {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(repo readers.MessageRepository, authz authz.Service, authn mainflux.ThingsServiceClient, port string, errs chan error, logger logger.Logger) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Cassandra reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, authz, authn, "cassandra-reader"))
}
