package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/vizee/ksrp/kube"
)

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

func main() {
	var (
		linkAddr  string
		apiAddr   string
		namespace string
		appName   string
		debug     bool
	)
	flag.StringVar(&linkAddr, "link", ":5777", "listen link")
	flag.StringVar(&apiAddr, "api", ":5780", "listen api")
	flag.StringVar(&namespace, "ns", "default", "kubernetes namespace")
	flag.StringVar(&appName, "app", "ksrp-expose", "kubernetes app label")
	flag.BoolVar(&debug, "debug", false, "debug log")
	flag.Parse()

	if debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	var operator *kube.ExposeOperator
	if os.Getenv("NO_HIJACK") != "1" {
		var err error
		operator, err = loadExposeOperator(namespace)
		if err != nil {
			fatal(err)
		}
	}

	server := newServer(appName, operator)

	slog.Info("listen link", "address", linkAddr)

	ln, err := net.Listen("tcp", linkAddr)
	if err != nil {
		fatal("listen link", err)
	}
	go server.serveAgent(ln)

	err = serveAPI(server, apiAddr)
	if err != nil {
		slog.Error("serve api", "err", err)
	}
}
