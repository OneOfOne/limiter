package limiter_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OneOfOne/limiter"
	"golang.org/x/xerrors"
)

func TestLimit(t *testing.T) {
	var (
		n  int64
		bg = limiter.NewWithContext(context.Background(), 2)
	)

	fn := func(ctx context.Context) error {
		atomic.AddInt64(&n, 1)
		<-ctx.Done()
		return nil
	}

	go func() {
		bg.Do(fn, nil)
		bg.Do(fn, nil)

		// fn will no longer run since when .Do executes, bg is cancele, nild
		bg.Do(fn, nil)
		bg.Do(fn, nil)
		atomic.StoreInt64(&n, 99)
	}()

	time.Sleep(time.Millisecond * 2)
	if nn := atomic.LoadInt64(&n); nn != 2 {
		t.Fatalf("expected 2, got %d", nn)
	}

	bg.Close()
	time.Sleep(time.Millisecond)

	if nn := atomic.LoadInt64(&n); nn != 99 {
		t.Fatalf("expected 99, got %d", nn)
	}

	bg = limiter.NewWithContext(context.Background(), 1)
	fn = func(ctx context.Context) error {
		atomic.AddInt64(&n, 1)
		time.Sleep(5 * time.Millisecond)
		return nil
	}

	errCh := make(chan error, 2)
	bg.DoWithTimeout(fn, time.Millisecond, errCh)
	select {
	case err := <-errCh:
		if !xerrors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected Deadline, got %v", err)
		}
	case <-time.After(time.Millisecond * 5):
		t.Fatal("didn't timeout :(")
	}

	if bg.IsCanceled() {
		t.Fatal("bg.IsCanceled() :( :(")
	}
}
