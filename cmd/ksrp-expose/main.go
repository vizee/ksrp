package main

import (
	"cmp"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/vizee/ksrp/kube"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Link      string     `yaml:"link"`
	API       string     `yaml:"api"`
	APIKey    string     `yaml:"apiKey"`
	Namespace string     `yaml:"namespace"`
	AppName   string     `yaml:"appName"`
	LogLevel  slog.Level `yaml:"logLevel"`
	NoHijack  bool       `yaml:"noHijack"`
}

const (
	operatorName = "ksrp-expose"
)

func loadExposeOperator(namespace string) (*kube.ExposeOperator, error) {
	client, err := kube.InClusterClient(operatorName)
	if err != nil {
		return nil, err
	}
	return kube.NewExposeOperator(operatorName, client, namespace), nil
}

func fatal(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func loadConfig(fname string) (*Config, error) {
	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func main() {
	conf, err := loadConfig(cmp.Or(os.Getenv("EXPOSE_CONFIG"), "config/expose.yaml"))
	if err != nil {
		fatal("load config", err)
	}

	slog.SetLogLoggerLevel(conf.LogLevel)

	var operator *kube.ExposeOperator
	if !conf.NoHijack {
		var err error
		operator, err = loadExposeOperator(conf.Namespace)
		if err != nil {
			fatal(err)
		}
	}

	server := newServer(conf.AppName, operator)

	slog.Info("listen link", "address", conf.Link)

	ln, err := net.Listen("tcp", conf.Link)
	if err != nil {
		fatal("listen link", err)
	}
	go server.serveAgent(ln)

	err = serveAPI(server, conf.API, conf.APIKey)
	if err != nil {
		slog.Error("serve api", "err", err)
	}
}
