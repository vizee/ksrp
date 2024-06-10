package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

func getPort(port string) ([]string, error) {
	resp, err := http.Get(getAPIUrl("/expose/port", url.Values{
		"port": []string{port},
	}))
	if err != nil {
		return nil, err
	}
	respData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API Error: %d %s", resp.StatusCode, string(respData))
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	return strings.Split(string(respData), "\n"), nil
}

func portCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "port port",
		Short: "Get port",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			portInfo, err := getPort(args[0])
			if err != nil {
				fatal("get port:", err)
			}
			if len(portInfo) == 0 {
				fmt.Println("port is not in use")
				return
			}
			for _, s := range portInfo {
				s = strings.TrimSpace(s)
				if s == "" {
					continue
				}
				fmt.Println(s)
			}
		},
	}
	return cmd
}
