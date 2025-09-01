package middleware

import (
	"aegis/internal/usecase"
	"context"
	"fmt"
	"sync"
)

// Chainer implements chain of responsibility pattern for middleware processing
type Chainer struct {
	ctx   context.Context
	chain []usecase.Middleware
	mutex sync.RWMutex // Thread safety for concurrent access
}

// Add middleware to the handling chain with validation
func (c *Chainer) Add(m usecase.Middleware) error {
	if m == nil {
		return fmt.Errorf("middleware cannot be nil")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.chain = append(c.chain, m)
	return nil
}

// Head returns the first middleware in the chain
func (c *Chainer) head() usecase.Middleware {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if len(c.chain) == 0 {
		return nil
	}
	return c.chain[0]
}

// Length returns the number of middleware in the chain
func (c *Chainer) Length() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.chain)
}

// Clear removes all middleware from the chain
func (c *Chainer) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.chain = c.chain[:0]
}

// Execute runs the entire middleware chain
func (c *Chainer) Execute(request *usecase.Request, response usecase.ResponseSender) error {
	head := c.head()
	if head == nil {
		return fmt.Errorf("no middleware in chain")
	}
	return head.Handle(request, response)
}

// NewChainer creates a new middleware chainer with context
func NewChainer(ctx context.Context) *Chainer {
	if ctx == nil {
		ctx = context.Background()
	}

	return &Chainer{
		ctx:   ctx,
		chain: make([]usecase.Middleware, 0),
		mutex: sync.RWMutex{},
	}
}
