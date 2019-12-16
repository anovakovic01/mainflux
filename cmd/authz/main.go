package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"

	"github.com/mainflux/mainflux/authz"
	cmodel "github.com/mainflux/mainflux/authz/casbin"

	natswatcher "github.com/Soluto/casbin-nats-watcher"
	xormadapter "github.com/casbin/xorm-adapter"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	authngrpc "github.com/mainflux/mainflux/authn/api/grpc"
	"github.com/mainflux/mainflux/authz/api"
	_ "github.com/mainflux/mainflux/authz/api/docs"
	authzgrpc "github.com/mainflux/mainflux/authz/api/grpc"
	httpapi "github.com/mainflux/mainflux/authz/api/http"
	pb "github.com/mainflux/mainflux/authz/api/pb"
	"github.com/mainflux/mainflux/authz/authn"
	"github.com/mainflux/mainflux/authz/local"
	"github.com/mainflux/mainflux/logger"
	broker "github.com/nats-io/go-nats"
	opentracing "github.com/opentracing/opentracing-go"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	jconfig "github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defLogLvl          = "error"
	defHTTPPort        = "8180"
	defGRPCPort        = "8181"
	defServerCert      = ""
	defServerKey       = ""
	defDBHost          = "localhost"
	defDBPort          = "5432"
	defDBUser          = ""
	defDBPass          = ""
	defNatsURL         = broker.DefaultURL
	defNatsSubject     = "authz"
	defSingleUserEmail = ""
	defSingleUserToken = ""
	defAuthnURL        = "http://localhost:8181"
	defAuthnTimeout    = "1" // in seconds
	defClientTLS       = "false"
	defCACerts         = ""
	defJaegerURL       = ""

	envLogLvl          = "MF_AUTHZ_LOG_LEVEL"
	envHTTPPort        = "MF_AUTHZ_HTTP_PORT"
	envGRPCPort        = "MF_AUTHZ_GRPC_PORT"
	envServerCert      = "MF_AUTHZ_SERVER_CERT"
	envServerKey       = "MF_AUTHZ_SERVER_KEY"
	envDBHost          = "MF_AUTHZ_DB_HOST"
	envDBPort          = "MF_AUTHZ_DB_PORT"
	envDBUser          = "MF_AUTHZ_DB_USER"
	envDBPass          = "MF_AUTHZ_DB_PASS"
	envNatsURL         = "MF_AUTHZ_NATS_URL"
	envNatsSubject     = "MF_AUTHZ_NATS_SUBJECT"
	envSingleUserEmail = "MF_AUTHZ_SINGLE_USER_EMAIL"
	envSingleUserToken = "MF_AUTHZ_SINGLE_USER_TOKEN"
	envAuthnURL        = "MF_AUTHN_URL"
	envAuthnTimeout    = "MF_AUTHZ_AUTHN_TIMEOUT"
	envClientTLS       = "MF_AUTHZ_CLIENT_TLS"
	envCACerts         = "MF_AUTHZ_CA_CERTS"
	envJaegerURL       = "MF_JAEGER_URL"
)

type config struct {
	logLvl          string
	httpPort        string
	grpcPort        string
	serverCert      string
	serverKey       string
	dbHost          string
	dbPort          string
	dbUser          string
	dbPass          string
	natsURL         string
	natsSubject     string
	singleUserEmail string
	singleUserToken string
	authnURL        string
	authnTimeout    time.Duration
	clientTLS       bool
	caCerts         string
	jaegerURL       string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLvl)
	if err != nil {
		log.Fatalf(err.Error())
	}

	tracer, closer := initJaeger("authz", cfg.jaegerURL, logger)
	defer closer.Close()

	watcher, err := natswatcher.NewWatcher(cfg.natsURL, cfg.natsSubject)
	if err != nil {
		log.Fatalf(err.Error())
	}

	connStr := fmt.Sprintf("user=%s password=%s host=%s port=%s sslmode=disable", cfg.dbUser, cfg.dbPass, cfg.dbHost, cfg.dbPort)
	adapter, err := xormadapter.NewAdapter("postgres", connStr)
	if err != nil {
		log.Fatalf(err.Error())
	}

	m, err := model.NewModelFromString(cmodel.Model)
	if err != nil {
		log.Fatalf(err.Error())
	}

	enf, err := casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		log.Fatalf(err.Error())
	}
	enf.EnableAutoSave(true)
	defer enf.SavePolicy()

	enf.SetWatcher(watcher)

	if err := enf.LoadPolicy(); err != nil {
		log.Fatalf(err.Error())
	}

	idp, close := createIdentityProvider(cfg, tracer, logger)
	defer close()

	svc := newService(enf, idp, logger)

	errs := make(chan error, 2)

	endpoints := api.New(svc, tracer)
	go startHTTPServer(endpoints, tracer, cfg, logger, errs)
	go startGRPCServer(endpoints, tracer, cfg, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("AuthZ service terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	timeout, err := strconv.ParseInt(mainflux.Env(envAuthnTimeout, defAuthnTimeout), 10, 64)
	if err != nil {
		log.Fatalf("Invalid %s value: %s", envAuthnTimeout, err.Error())
	}

	return config{
		logLvl:          mainflux.Env(envLogLvl, defLogLvl),
		httpPort:        mainflux.Env(envHTTPPort, defHTTPPort),
		grpcPort:        mainflux.Env(envGRPCPort, defGRPCPort),
		serverCert:      mainflux.Env(envServerCert, defServerCert),
		serverKey:       mainflux.Env(envServerKey, defServerKey),
		dbHost:          mainflux.Env(envDBHost, defDBHost),
		dbPort:          mainflux.Env(envDBPort, defDBPort),
		dbUser:          mainflux.Env(envDBUser, defDBUser),
		dbPass:          mainflux.Env(envDBPass, defDBPass),
		natsURL:         mainflux.Env(envNatsURL, defNatsURL),
		natsSubject:     mainflux.Env(envNatsSubject, defNatsSubject),
		singleUserEmail: mainflux.Env(envSingleUserEmail, defSingleUserEmail),
		singleUserToken: mainflux.Env(envSingleUserToken, defSingleUserToken),
		authnURL:        mainflux.Env(envAuthnURL, defAuthnURL),
		authnTimeout:    time.Duration(timeout) * time.Second,
		clientTLS:       tls,
		caCerts:         mainflux.Env(envCACerts, defCACerts),
		jaegerURL:       mainflux.Env(envJaegerURL, defJaegerURL),
	}
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
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		os.Exit(1)
	}

	return tracer, closer
}

func createIdentityProvider(cfg config, tracer opentracing.Tracer, logger logger.Logger) (authz.IdentityProvider, func() error) {
	if cfg.singleUserEmail != "" && cfg.singleUserToken != "" {
		return local.NewIDP(cfg.singleUserEmail, cfg.singleUserToken), nil
	}

	conn := connectToAuthN(cfg, logger)
	authnClient := authngrpc.NewClient(tracer, conn, cfg.authnTimeout)
	return authn.New(authnClient), conn.Close
}

func connectToAuthN(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to create tls credentials: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		opts = append(opts, grpc.WithInsecure())
		logger.Info("gRPC communication is not encrypted")
	}

	conn, err := grpc.Dial(cfg.authnURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to the authn service: %s", err))
		os.Exit(1)
	}

	return conn
}

func newService(enf authz.Enforcer, idp authz.IdentityProvider, logger logger.Logger) authz.Service {
	svc := authz.New(enf, idp)
	svc = api.LoggingMiddleware(svc, logger)
	api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "authz",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "authz",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(svc api.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger, errs chan error) {
	handler := httpapi.MakeHandler(svc, logger)

	p := fmt.Sprintf(":%s", cfg.httpPort)
	if cfg.serverCert != "" || cfg.serverKey != "" {
		logger.Info(fmt.Sprintf("AuthZ service started using https on port %s with cert %s key %s",
			cfg.httpPort, cfg.serverCert, cfg.serverKey))
		errs <- http.ListenAndServeTLS(p, cfg.serverCert, cfg.serverKey, handler)
		return
	}

	logger.Info(fmt.Sprintf("AuthZ service started using http on port %s", cfg.httpPort))
	errs <- http.ListenAndServe(p, handler)
}

func startGRPCServer(svc api.Service, tracer opentracing.Tracer, cfg config, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", cfg.grpcPort)
	listener, err := net.Listen("tcp", p)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to listen on port %s: %s", cfg.grpcPort, err))
		os.Exit(1)
	}

	var server *grpc.Server
	if cfg.serverCert != "" || cfg.serverKey != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.serverCert, cfg.serverKey)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load AuthZ certificates: %s", err))
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("AuthZ gRPC service started using https on port %s with cert %s key %s",
			cfg.grpcPort, cfg.serverCert, cfg.serverKey))
		server = grpc.NewServer(grpc.Creds(creds))
	} else {
		logger.Info(fmt.Sprintf("Things gRPC service started using http on port %s", cfg.grpcPort))
		server = grpc.NewServer()
	}

	pb.RegisterAuthZServiceServer(server, authzgrpc.NewServer(svc))
	errs <- server.Serve(listener)
}
