package httpserver

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
)

type Server struct {
	server *http.Server
}

func NewServer(server *http.Server) *Server {
	return &Server{
		server,
	}
}

func (s Server) Start() error {
	err := s.server.ListenAndServe()
	if errors.Cause(err) == http.ErrServerClosed {
		return nil
	}

	return err
}

func (s Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
