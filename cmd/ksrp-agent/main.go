package main

import (
	"bufio"
	"bytes"
	"cmp"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	appName = "ksrp-agent"
)

var (
	apiAddress  string = os.Getenv("KSRP_API")
	apiKey      string = os.Getenv("KSRP_APIKEY")
	linkAddress string = os.Getenv("KSRP_LINK")
)

func fatal(args ...any) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(1)
}

func loadConfigFromFile() (map[string]string, error) {
	const defaultConfigFile = "agent.conf"

	confDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(confDir, "ksrp", defaultConfigFile))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	conf := make(map[string]string)
	rd := bufio.NewReader(f)
	lineNo := 0
	for {
		line, err := rd.ReadSlice('\n')
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			if len(line) == 0 {
				break
			}
		}

		lineNo++
		comment := bytes.IndexByte(line, '#')
		if comment >= 0 {
			line = line[:comment]
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		expr := string(line)
		eq := strings.IndexByte(expr, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("invalid line: %d", lineNo)
		}

		conf[expr[:eq]] = expr[eq+1:]
	}

	return conf, nil
}

func main() {
	var (
		debug bool
	)
	app := cobra.Command{
		Use: appName,
		PersistentPreRunE: func(_ *cobra.Command, args []string) error {
			config, err := loadConfigFromFile()
			if err == nil {
				apiAddress = cmp.Or(apiAddress, config["api"])
				apiKey = cmp.Or(linkAddress, config["api_key"])
				linkAddress = cmp.Or(linkAddress, config["link"])
			} else if !os.IsNotExist(err) {
				slog.Warn("load config", "err", err)
			}
			if !strings.Contains(apiAddress, "://") {
				apiAddress = "http://" + apiAddress
			}
			if debug {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}
			return nil
		},
	}
	app.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode")
	app.AddCommand(listenCommand(), linkCommand(), revokeCommand(), portCommand())
	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
