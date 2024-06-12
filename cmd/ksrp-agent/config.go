package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func getDefaultConfigPath() string {
	const defaultConfigFile = "agent.conf"

	confDir, err := os.UserConfigDir()
	if err != nil {
		return "./config/ksrp/" + defaultConfigFile
	}
	return filepath.Join(confDir, "ksrp", defaultConfigFile)
}

func loadConfig(fname string) (map[string]string, error) {
	f, err := os.Open(fname)
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

func writeConfig(fname string, values map[string]string) error {
	err := os.MkdirAll(filepath.Dir(fname), 0755)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	for k, v := range values {
		_, _ = buf.WriteString(k)
		_ = buf.WriteByte('=')
		_, _ = buf.WriteString(v)
		_ = buf.WriteByte('\n')
	}
	return os.WriteFile(fname, buf.Bytes(), 0644)
}

func saveConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "save-config",
		Short: "Save configuration",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			err := writeConfig(getDefaultConfigPath(), map[string]string{
				"api":     apiAddress,
				"api_key": apiKey,
				"link":    linkAddress,
			})
			if err != nil {
				fatal(err)
			}
			fmt.Printf("save configuration to %s\n", getDefaultConfigPath())
		},
	}
}
