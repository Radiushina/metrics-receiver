package main

import (
	"context"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 10 * time.Second
	idleTimeout       = 60 * time.Second
)

// Server оборачивает http.Server с настроенными таймаутами чтения и записи.
type Server struct {
	httpServer *http.Server
}

// Run запускает HTTP-сервер с указанным обработчиком на адресе из флага -a.
func (s *Server) Run(handler http.Handler) error {
	s.httpServer = &http.Server{
		Addr:              flagRunAddr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown корректно останавливает HTTP-сервер, дожидаясь завершения активных запросов.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
