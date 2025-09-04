package server

import (
	"aegis/internal/middleware"
	"aegis/internal/usecase"
	"context"
	"fmt"
	"log/slog"
	"os"
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
	var chainer *middleware.Chainer
	switch verificationType {
	case "js-challenge":
		chainer = middleware.DefaultProtectionChain(server.ctx, verificationComplexity, protections)
	case "captcha":
		chainer = middleware.CaptchaProtectionChain(server.ctx, verificationComplexity, protections, "/home/ash/projects/aegis/assets/classification_captcha/templates.json")
	default:
		slog.Error("Unable to start server", slog.String("error", fmt.Sprintf("unknown type: %s", verificationType)))
		os.Exit(1)
	}
	server.api = NewApiServer(address, chainer)

	return &server
}
