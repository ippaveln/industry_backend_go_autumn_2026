package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func TestIndependentErrors(t *testing.T) {
	boom := errors.New("boom")
	calls := 0
	r, e := ParallelMap(context.Background(), 1, []int{1, 2, 3}, func(_ context.Context, n int) (int, error) {
		calls++
		if n == 1 {
			return 99, boom
		}
		return n * 2, nil
	})
	if e != nil || len(r) != 3 || calls != 3 {
		t.Fatalf("must process all: %v %v %d", r, e, calls)
	}
	if r[0].Value != 0 || !errors.Is(r[0].Err, boom) || r[1].Value != 4 || r[2].Value != 6 || r[1].Err != nil {
		t.Fatal(r)
	}
}
func TestEdges(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fn := func(context.Context, int) (int, error) { t.Error("must not call"); return 0, nil }
	for _, w := range []int{-1, 0} {
		if r, e := ParallelMap(ctx, w, []int{1}, fn); r != nil || !errors.Is(e, ErrInvalidWorkers) {
			t.Fatal(r, e)
		}
	}
	if r, e := ParallelMap(ctx, 1, []int{}, fn); r != nil || !errors.Is(e, context.Canceled) {
		t.Fatal(r, e)
	}
	if r, e := ParallelMap(context.Background(), 1, []int{}, fn); len(r) != 0 || e != nil {
		t.Fatal(r, e)
	}
}
func TestBoundedOrderAndJoin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const n = 20
		in := make([]int, n)
		for i := range in {
			in[i] = i
		}
		gates := make([]chan struct{}, n)
		for i := range gates {
			gates[i] = make(chan struct{})
		}
		var active, peak, calls atomic.Int32
		done := make(chan struct{})
		var out []Result[int]
		var err error
		go func() {
			defer close(done)
			out, err = ParallelMap(context.Background(), 3, in, func(_ context.Context, n int) (int, error) {
				a := active.Add(1)
				for p := peak.Load(); a > p; p = peak.Load() {
					if peak.CompareAndSwap(p, a) {
						break
					}
				}
				calls.Add(1)
				<-gates[n]
				active.Add(-1)
				return n * 2, nil
			})
		}()
		synctest.Wait()
		if calls.Load() != 3 {
			t.Fatalf("expected 3 blocked workers, got %d", calls.Load())
		}
		for i := n - 1; i >= 0; i-- {
			close(gates[i])
		}
		<-done
		if err != nil || len(out) != n || peak.Load() > 3 || active.Load() != 0 {
			t.Fatalf("%v peak=%d active=%d", err, peak.Load(), active.Load())
		}
		for i := range out {
			if out[i].Value != i*2 || out[i].Err != nil {
				t.Fatal("order", i, out[i])
			}
		}
	})
}
func TestCancelWaitsForWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var started, ended atomic.Int32
		done := make(chan struct{})
		go func() {
			defer close(done)
			r, e := ParallelMap(ctx, 2, []int{1, 2, 3, 4}, func(ctx context.Context, n int) (int, error) {
				started.Add(1)
				<-ctx.Done()
				ended.Add(1)
				return 0, ctx.Err()
			})
			if r != nil || !errors.Is(e, context.Canceled) {
				t.Error(r, e)
			}
		}()
		synctest.Wait()
		if started.Load() != 2 {
			t.Fatal("pool did not fill")
		}
		cancel()
		<-done
		if ended.Load() != started.Load() {
			t.Fatal("workers remain")
		}
	})
}
