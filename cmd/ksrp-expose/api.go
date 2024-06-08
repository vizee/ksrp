package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

type apiServer struct {
	inner *Server
}

func (s *apiServer) postListen(w http.ResponseWriter, r *http.Request) {
	service := r.FormValue("service")
	port, _ := strconv.Atoi(r.FormValue("port"))
	if port <= 0 {
		http.Error(w, "invalid port", http.StatusBadRequest)
		return
	}

	slog.Info("listen service", "name", service, "port", port)

	svc, err := s.inner.listenService(service, port)
	if err != nil {
		slog.Error("listen service", "port", port, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("hijack service", "name", service, "token", svc.token)

	err = s.inner.hijackService(r.Context(), service, port)
	if err != nil {
		slog.Error("hijack service", "service", service, "port", port, "err", err)

		// 尝试释放监听
		err2 := s.inner.revokeToken(context.Background(), svc.token, false)
		if err2 != nil {
			slog.Warn("revoke token", "err", err2)
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(svc.token))
}

func (s *apiServer) postRevoke(w http.ResponseWriter, r *http.Request) {
	err := s.inner.revokeToken(r.Context(), r.FormValue("token"), true)
	if err != nil {
		slog.Warn("revoke token", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("ok"))
}

func (s *apiServer) getPort(w http.ResponseWriter, r *http.Request) {
	port, _ := strconv.Atoi(r.FormValue("port"))
	svc := s.inner.getPort(port)
	if svc == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	fmt.Fprintf(w, "%s\n%s\n", svc.token, svc.name)
}

func (s *apiServer) getHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("ok"))
}

func serveAPI(server *Server, address string) error {
	api := &apiServer{inner: server}
	http.HandleFunc("POST /expose/listen", api.postListen)
	http.HandleFunc("POST /expose/revoke", api.postRevoke)
	http.HandleFunc("GET /expose/port", api.getPort)
	http.HandleFunc("GET /-/healthz", api.getHealthz)

	slog.Info("listen API", "address", address)

	return http.ListenAndServe(address, nil)
}
