package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	mathrand "math/rand/v2"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vizee/ksrp/ioutil"
	"github.com/vizee/ksrp/kube"
	"github.com/vizee/ksrp/proto"
	"github.com/vizee/mstp"
)

var (
	errBadShakeHands = errors.New("unexpected sha")
)

type Service struct {
	ln    net.Listener
	token string
	name  string
	port  int

	closed atomic.Bool
	signal chan struct{}
	acs    []*mstp.Conn
	lock   sync.Mutex
}

func (s *Service) close() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	s.ln.Close()

	s.lock.Lock()
	acs := s.acs
	s.acs = nil
	s.lock.Unlock()

	for _, ac := range acs {
		ac.Close()
	}
}

func (s *Service) removeAgentConn(msc *mstp.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	for i, ac := range s.acs {
		if ac == msc {
			lastIdx := len(s.acs) - 1
			last := s.acs[lastIdx]
			s.acs[lastIdx] = nil
			s.acs[i] = last
			s.acs = s.acs[:lastIdx]
			break
		}
	}
}

func (s *Service) addAgentConn(ac *mstp.Conn) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.closed.Load() {
		return false
	}

	s.acs = append(s.acs, ac)

	return true
}

func (s *Service) getAgentConn() (*mstp.Conn, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.acs) == 0 {
		return nil, false
	}

	return s.acs[mathrand.IntN(len(s.acs))], true
}

func generateToken() string {
	var rnd [18]byte
	_, _ = rand.Read(rnd[:])
	return base64.RawURLEncoding.EncodeToString(rnd[:])
}

type Server struct {
	appName  string
	operator *kube.ExposeOperator
	ports    map[int]*Service
	tokens   map[string]*Service
	lock     sync.RWMutex
}

func (s *Server) handleServiceConn(svc *Service, sc net.Conn) {
	defer sc.Close()

	ac, ok := svc.getAgentConn()
	if !ok {
		slog.Debug("no agent connection available", "service", svc.name)
		return
	}
	as, err := ac.NewStream()
	if err != nil {
		slog.Error("new agent stream", "ac", fmt.Sprintf("%p", ac), "err", err)
		return
	}
	defer as.Close()

	slog.Debug("copy traffic", "sc", sc.RemoteAddr().String(), "ac", fmt.Sprintf("%p", ac))
	err = ioutil.DualCopy(sc, as)
	if err != nil && err != io.EOF {
		slog.Error("copy traffic", "sc", sc.RemoteAddr().String(), "ac", fmt.Sprintf("%p", ac), "err", err)
	}
}

func (s *Server) serveService(svc *Service) {
	for {
		sc, err := svc.ln.Accept()
		if err != nil {
			if svc.closed.Load() {
				return
			}
			slog.Warn("accept service connection", "port", svc.port, "err", err)
			time.Sleep(time.Second)
			continue
		}

		slog.Debug("new service connection", "port", svc.port)

		go s.handleServiceConn(svc, sc)
	}
}

func (s *Server) listenService(service string, port int) (*Service, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}

	token := generateToken()
	svc := &Service{
		ln:     ln,
		token:  token,
		name:   service,
		port:   port,
		signal: make(chan struct{}, 1),
	}

	// 如果能 listen 成功，那么 port 必然不会冲突，以及忽略 token 冲突的情况
	s.lock.Lock()
	s.ports[port] = svc
	s.tokens[token] = svc
	s.lock.Unlock()

	go s.serveService(svc)

	return svc, nil
}

func (s *Server) getPort(port int) *Service {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.ports[port]
}

func (s *Server) hijackService(ctx context.Context, serviceName string, port int) error {
	if s.operator == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.operator.HijackService(ctx, serviceName, s.appName, port)
}

func (s *Server) revokeToken(ctx context.Context, token string, restore bool) error {
	slog.Info("revoke token", "token", token)

	s.lock.Lock()
	svc := s.tokens[token]
	if svc != nil {
		delete(s.tokens, token)
		delete(s.ports, svc.port)
	}
	s.lock.Unlock()
	if svc == nil {
		slog.Debug("invalid token", "token", token)
		return nil
	}

	if s.operator != nil && restore {
		slog.Info("restore service", "name", svc.name)

		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := s.operator.RestoreService(ctx, svc.name)
		cancel()
		if err != nil {
			slog.Warn("restore service", "name", svc.name)
		}
	}

	slog.Info("close service", "name", svc.name, "token", svc.token)

	svc.close()

	return nil
}

func (s *Server) handleAgentConn(conn net.Conn) {
	defer conn.Close()

	svc, err := s.shakeHandsWithAgent(conn)
	if err != nil || svc == nil {
		slog.Debug("agent shake hands", "conn", conn.RemoteAddr().String(), "err", err)
		return
	}

	slog.Debug("service add agent connection", "name", svc.name, "conn", conn.RemoteAddr().String())

	msc := mstp.NewConn(conn, true, nil)
	defer msc.Close()
	if !svc.addAgentConn(msc) {
		return
	}
	err = msc.LastErr()
	if err != nil && err != io.EOF {
		slog.Error("agent connection error", "conn", conn.RemoteAddr().String(), "err", err)
	}
	svc.removeAgentConn(msc)
}

func (s *Server) shakeHandsWithAgent(conn net.Conn) (*Service, error) {
	conn.SetReadDeadline(time.Now().Add(time.Second))
	cmd, token, err := proto.ReadMessage(conn)
	conn.SetReadDeadline(time.Time{})
	if err != nil {
		return nil, err
	}
	if cmd != proto.CmdShakeHands {
		return nil, errBadShakeHands
	}

	s.lock.RLock()
	svc := s.tokens[token]
	s.lock.RUnlock()

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	if svc == nil {
		_ = proto.WriteMessage(conn, proto.CmdError, "invalid token")
	} else {
		_ = proto.WriteMessage(conn, proto.CmdShakeHandsOk, "ok")
	}
	conn.SetWriteDeadline(time.Time{})

	return svc, nil
}

func (s *Server) serveAgent(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Warn("accept agent connection", "err", err)
			time.Sleep(time.Second)
			continue
		}

		slog.Debug("new agent connection", "conn", conn.RemoteAddr().String())

		go s.handleAgentConn(conn)
	}
}

func newServer(appName string, operator *kube.ExposeOperator) *Server {
	return &Server{
		appName:  appName,
		operator: operator,
		ports:    make(map[int]*Service),
		tokens:   make(map[string]*Service),
	}
}
