package middleware

import "aegis/internal/usecase"

type Middleware[T any] interface {
	Handle(*usecase.RequestContext[T], ResponseSender)
	Bind(Middleware[T])
}

type ResponseSender interface {
	Allow()
	Deny()
}
