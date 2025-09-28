package middleware

import (
	"aegis/internal/usecase"
	"fmt"
)

type Chain[T any] struct {
	chain []Middleware[T]
}

// Execute runs the entire middleware chain
func (c *Chain[T]) Execute(request *usecase.RequestContext[T], response ResponseSender) error {
	if len(c.chain) == 0 {
		return fmt.Errorf("chain is empty")
	}
	c.chain[0].Handle(request, response)
	return nil
}

func NewChain[T any](middlewares ...Middleware[T]) *Chain[T] {
	chain := Chain[T]{}
	chain.chain = middlewares
	for i := 0; i < len(middlewares)-1; i++ {
		chain.chain[i].Bind(chain.chain[i+1])
	}
	return &chain
}
