package server

import (
	"aegis/internal/middleware"
	"aegis/internal/usecase"
	"context"
)

type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	api    *ApiServer
}

func (s *Server) Serve() (err error) {
	s.api.Serve()
	return
}

func (s *Server) Shutdown(ctx context.Context) (err error) {
	return s.api.Shutdown(ctx)
}

func NewServer(
	appCtx context.Context,
	address string,
	protections []usecase.Protection,
	verificationType string,
	verificationComplexity string,
) *Server {
	server := Server{}
	server.ctx, server.cancel = context.WithCancel(appCtx)

	chain := middleware.DefaultProtectionChain(server.ctx, verificationComplexity, protections)
	server.api = NewApiServer(address, chain)

	return &server
}
