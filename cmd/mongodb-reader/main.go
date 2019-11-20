// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authz"
	authzgrpc "github.com/mainflux/mainflux/authz/api/grpc"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/mongodb"
	thingsapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defThingsURL     = "localhost:8181"
	defLogLevel      = "error"
	defPort          = "8180"
	defDBName        = "mainflux"
	defDBHost        = "localhost"
	defDBPort        = "27017"
	defClientTLS     = "false"
	defCACerts       = ""
	defJaegerURL     = ""
	defThingsTimeout = "1" // in seconds
	defAuthzURL      = "localhost:8181"

	envThingsURL     = "MF_THINGS_URL"
	envLogLevel      = "MF_MONGO_READER_LOG_LEVEL"
	envPort          = "MF_MONGO_READER_PORT"
	envDBName        = "MF_MONGO_READER_DB_NAME"
	envDBHost        = "MF_MONGO_READER_DB_HOST"
	envDBPort        = "MF_MONGO_READER_DB_PORT"
	envClientTLS     = "MF_MONGO_READER_CLIENT_TLS"
	envCACerts       = "MF_MONGO_READER_CA_CERTS"
	envJaegerURL     = "MF_JAEGER_URL"
	envThingsTimeout = "MF_MONGO_READER_THINGS_TIMEOUT"
	envAuthzURL      = "MF_AUTHZ_URL"
)

type config struct {
	thingsURL     string
	logLevel      string
	port          string
	dbName        string
	dbHost        string
	dbPort        string
	clientTLS     bool
	caCerts       string
	jaegerURL     string
	thingsTimeout time.Duration
	authzURL      string
}

func main() {
	cfg := loadConfigs()
	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	thingsConn := connectToGRPC(cfg, cfg.thingsURL, logger)
	defer thingsConn.Close()

	thingsTracer, thingsCloser := initJaeger("things", cfg.jaegerURL, logger)
	defer thingsCloser.Close()

	authn := thingsapi.NewClient(thingsConn, thingsTracer, cfg.thingsTimeout)

	authzConn := connectToGRPC(cfg, cfg.authzURL, logger)
	defer authzConn.Close()

	authzTracer, authzCloser := initJaeger("authz", cfg.jaegerURL, logger)
	defer authzCloser.Close()

	authz := authzgrpc.NewClient(authzConn, authzTracer)

	db := connectToMongoDB(cfg.dbHost, cfg.dbPort, cfg.dbName, logger)

	repo := newService(db, logger)

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPServer(repo, authz, authn, cfg.port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("MongoDB reader service terminated: %s", err))
}

func loadConfigs() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	timeout, err := strconv.ParseInt(mainflux.Env(envThingsTimeout, defThingsTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envThingsTimeout, err.Error())
	}

	return config{
		thingsURL:     mainflux.Env(envThingsURL, defThingsURL),
		logLevel:      mainflux.Env(envLogLevel, defLogLevel),
		port:          mainflux.Env(envPort, defPort),
		dbName:        mainflux.Env(envDBName, defDBName),
		dbHost:        mainflux.Env(envDBHost, defDBHost),
		dbPort:        mainflux.Env(envDBPort, defDBPort),
		clientTLS:     tls,
		caCerts:       mainflux.Env(envCACerts, defCACerts),
		jaegerURL:     mainflux.Env(envJaegerURL, defJaegerURL),
		thingsTimeout: time.Duration(timeout) * time.Second,
		authzURL:      mainflux.Env(envAuthzURL, defAuthzURL),
	}
}

func connectToMongoDB(host, port, name string, logger logger.Logger) *mongo.Database {
	addr := fmt.Sprintf("mongodb://%s:%s", host, port)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to database: %s", err))
		os.Exit(1)
	}

	return client.Database(name)
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

func newService(db *mongo.Database, logger logger.Logger) readers.MessageRepository {
	repo := mongodb.New(db)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "mongodb",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "mongodb",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(repo readers.MessageRepository, authz authz.Service, authn mainflux.ThingsServiceClient, port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Mongo reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, authz, authn, "mongodb-reader"))
}
