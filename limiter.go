package limiter

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"
)

// New returns a new runner with no limits.
func New() *Limiter { return NewWithContext(context.Background(), 0) }

// NewWithContext returns a new runner with the given parent context and limit, if limit is <= 0 it won't have a limit.
func NewWithContext(ctx context.Context, limit int) *Limiter {
	var bg Limiter
	bg.ctx, bg.cancel = context.WithCancel(ctx)

	if limit > 0 {
		bg.ch = make(chan struct{}, limit)
	}

	return &bg
}

type Limiter struct {
	wg sync.WaitGroup
	ch chan struct{}

	ctx    context.Context
	cancel func()
}

func (bg *Limiter) Add(fn func(ctx context.Context) error) <-chan error {
	errChan := make(chan error, 1)

	if !bg.add() {
		errChan <- context.Canceled
		close(errChan)
		return errChan
	}

	ch := make(chan error, 1)
	go func() { ch <- fn(bg.ctx); close(ch) }()

	go func() {
		select {
		case err := <-ch:
			errChan <- err
		case <-bg.ctx.Done():
			errChan <- xerrors.Errorf("global ctx: %w", bg.ctx.Err())
		}
		close(errChan)
		bg.done()
	}()

	return errChan
}

func (bg *Limiter) AddWithTimeout(fn func(ctx context.Context) error, timeout time.Duration) <-chan error {
	errChan := make(chan error, 1)

	if !bg.add() {
		errChan <- context.Canceled
		close(errChan)
		return errChan
	}

	tctx, cancel := context.WithTimeout(bg.ctx, timeout)

	ch := make(chan error, 1)
	go func() { ch <- fn(tctx); close(ch) }()

	go func() {
		select {
		case err := <-ch:
			errChan <- err
		case <-bg.ctx.Done():
			errChan <- xerrors.Errorf("global ctx: %w", bg.ctx.Err())
		case <-tctx.Done():
			errChan <- xerrors.Errorf("timeout ctx: %w", tctx.Err())

		}
		close(errChan)
		cancel()
		bg.done()
	}()

	return errChan
}

func (bg *Limiter) Context() context.Context { return bg.ctx }

func (bg *Limiter) Close() error {
	err := bg.ctx.Err()
	bg.cancel()
	return err
}

func (bg *Limiter) IsCanceled() bool {
	return bg.ctx.Err() != nil
}

func (bg *Limiter) Wait() { bg.wg.Wait() }
func (bg *Limiter) WaitWithContext(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		bg.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-bg.ctx.Done():
		return xerrors.Errorf("global ctx: %w", bg.ctx.Err())
	case <-ctx.Done():
		return xerrors.Errorf("ctx: %w", bg.ctx.Err())
	}
}

func (bg *Limiter) add() bool {
	if bg.ch != nil {
		bg.ch <- struct{}{}
	}

	if bg.IsCanceled() {
		return false
	}

	bg.wg.Add(1)
	return true
}

func (bg *Limiter) done() {
	if bg.ch != nil {
		<-bg.ch
	}

	bg.wg.Done()
}
