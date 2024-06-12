package main

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
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

func main() {
	var (
		logLevel string
	)
	app := cobra.Command{
		Use: appName,
		PersistentPreRunE: func(_ *cobra.Command, args []string) error {
			config, err := loadConfig(getDefaultConfigPath())
			if err == nil {
				apiAddress = cmp.Or(apiAddress, config["api"])
				apiKey = cmp.Or(apiKey, config["api_key"])
				linkAddress = cmp.Or(linkAddress, config["link"])
			} else if !os.IsNotExist(err) {
				slog.Warn("load config", "err", err)
			}
			if !strings.Contains(apiAddress, "://") {
				apiAddress = "http://" + apiAddress
			}
			var lv slog.Level
			if err := lv.UnmarshalText([]byte(logLevel)); err != nil {
				fatal(err)
			}
			slog.SetLogLoggerLevel(lv)
			return nil
		},
	}
	app.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug/info/warn/error")
	app.AddCommand(
		listenCommand(),
		linkCommand(),
		revokeCommand(),
		portCommand(),
		saveConfigCommand())
	err := app.Execute()
	if err != nil {
		os.Exit(1)
	}
}
