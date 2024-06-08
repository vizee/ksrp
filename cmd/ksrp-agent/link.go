package main

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vizee/ksrp/ioutil"
	"github.com/vizee/ksrp/proto"
	"github.com/vizee/mstp"
)

func shakeHandsWithExpose(conn net.Conn, token string) error {
	err := proto.WriteMessage(conn, proto.CmdShakeHands, token, time.Second)
	if err != nil {
		return err
	}
	cmd, msg, err := proto.ReadMessage(conn, time.Second*5)
	if err != nil {
		return err
	}
	switch cmd {
	case proto.CmdShakeHandsOk:
		return nil
	case proto.CmdError:
		return fmt.Errorf("error: %s", msg)
	default:
		return fmt.Errorf("unexpected cmd: %x", cmd)
	}
}

type linkOptions struct {
	backendConns int
}

func linkMain(token string, backend string, opts *linkOptions) {
	conn, err := net.Dial("tcp", linkAddress)
	if err != nil {
		slog.Error("dial link", "err", err)
		os.Exit(1)
	}
	err = shakeHandsWithExpose(conn, token)
	if err != nil {
		slog.Error("shake hands", "err", err)
		os.Exit(1)
	}

	backendPool := startLocalPool(backend, opts.backendConns)
	msc := mstp.NewConn(conn, false, func(s *mstp.Stream) {
		go func(s *mstp.Stream) {
			defer s.Close()
			bc, err := backendPool.get()
			if err != nil {
				slog.Error("get backend", "err", err)
				return
			}

			slog.Debug("copy traffic", "stream", fmt.Sprintf("%p", s), "backend", bc.RemoteAddr().String())

			err = ioutil.DualCopy(s, bc)
			if err != nil && err != io.EOF {
				slog.Error("copy traffic", "stream", fmt.Sprintf("%p", s), "backend", bc.RemoteAddr().String(), "err", err)
			}
		}(s)
	})
	defer msc.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	sig := <-signals
	slog.Info("stop", "signal", sig.String())
}

func linkCommand() *cobra.Command {
	var opts linkOptions
	cmd := &cobra.Command{
		Use:   "link token backend",
		Short: "Link expose with backend",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			linkMain(args[0], args[1], &opts)
		},
	}
	cmd.Flags().IntVar(&opts.backendConns, "backend-conns", 1, "backend conns")
	return cmd
}
