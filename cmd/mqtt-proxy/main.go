package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
)

const (
	envPort      = "MF_MQTT_ADAPTER_PORT"
	envLogLevel  = "MF_HTTP_ADAPTER_LOG_LEVEL"
	envBrokerURL = "MF_MQTT_BROKER_URL"
	defPort      = "8180"
	defLogLevel  = "error"
	defBrokerURL = "localhost:1883"
)

type config struct {
	port      string
	logLevel  string
	brokerURL string
}

func main() {
	conf := loadConfig()

	logger, err := logger.New(os.Stdout, conf.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", conf.port))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start MQTT server: %s", err))
		os.Exit(1)
	}
	defer ln.Close()

}

func loadConfig() config {
	return config{
		port:      mainflux.Env(envPort, defPort),
		logLevel:  mainflux.Env(envLogLevel, defLogLevel),
		brokerURL: mainflux.Env(envBrokerURL, defBrokerURL),
	}
}
