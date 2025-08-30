package service

import (
	"aegis/internal/usecase"
	"context"
)

type Chainer struct {
	ctx   context.Context
	chain []usecase.Middleware
}

// Add middleware to the handling chain
func (c *Chainer) Add(m usecase.Middleware) {
	c.chain = append(c.chain, m)
}

// Return a middleware in the head
func (c *Chainer) Head() usecase.Middleware {
	if len(c.chain) == 0 {
		return nil
	}
	return c.chain[0]
}

func NewChainer(ctx context.Context) *Chainer {
	return &Chainer{
		ctx:   ctx,
		chain: make([]usecase.Middleware, 0),
	}
}
