package screenshot

import (
	"context"
	"sync"
	"sync/atomic"
)

// WindowHooks bridges native window operations owned by the application.
type WindowHooks struct {
	Handle       func() (uintptr, error)
	SavePosition func() error
	Prepare      func() error
	Restore      func() error
}
type Service struct {
	ctx     context.Context
	mu      *sync.Mutex
	hooks   WindowHooks
	capture *screenshotSession
	active  atomic.Bool
}

func New(ctx context.Context, mu *sync.Mutex, hooks WindowHooks) *Service {
	return &Service{ctx: ctx, mu: mu, hooks: hooks}
}
func (s *Service) Active() bool { return s != nil && s.active.Load() }
