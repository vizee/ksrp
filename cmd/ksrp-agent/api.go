package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

func postRevoke(token string) (string, error) {
	resp, err := http.PostForm(getAPIUrl("/expose/revoke", nil), url.Values{
		"token": []string{token},
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
		return "", fmt.Errorf("API Error: %d %s", resp.StatusCode, string(respData))
	}
	return string(bytes.TrimSpace(respData)), nil
}

func revokeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke token",
		Short: "Revoke token",
		Args:  cobra.ExactArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			_, err := postRevoke(args[0])
			if err != nil {
				fatal("revoke token:", err)
			}
		},
	}
	return cmd
}
