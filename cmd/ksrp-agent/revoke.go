package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
)

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
