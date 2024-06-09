package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

func getAPIUrl(path string, values url.Values) string {
	if apiKey != "" {
		if values == nil {
			values = make(url.Values)
		}
		values.Set("key", apiKey)
	}
	if len(values) != 0 {
		return apiAddress + path + "?" + values.Encode()
	} else {
		return apiAddress + path
	}
}

func postListen(port string, service string) (string, error) {
	resp, err := http.PostForm(getAPIUrl("/expose/listen", nil), url.Values{
		"service": []string{service},
		"port":    []string{port},
	})
	if err != nil {
		return "", err
	}
	respData, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API Error: %d %s", resp.StatusCode, string(bytes.TrimSpace(respData)))
	}
	return string(bytes.TrimSpace(respData)), nil
}

func listenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "listen port service",
		Short: "Listen service",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			token, err := postListen(args[0], args[1])
			if err != nil {
				fatal("listen service:", err)
			}
			fmt.Println(token)
		},
	}
	return cmd
}
