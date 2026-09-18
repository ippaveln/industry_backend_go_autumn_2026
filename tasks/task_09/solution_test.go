package main

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestIndependentErrors(t *testing.T) {
	boom := errors.New("boom")
	var calls atomic.Int32
	fn := func(_ context.Context, n int) (int, error) {
		calls.Add(1)
		if n == 1 {
			return 99, boom
		}
		return n * 2, nil
	}

	got, err := ParallelMap(context.Background(), 1, []int{1, 2, 3}, fn)

	if err != nil || len(got) != 3 || calls.Load() != 3 {
		t.Fatalf("must process all items: results=%v err=%v calls=%d", got, err, calls.Load())
	}
	if got[0].Value != 0 || !errors.Is(got[0].Err, boom) {
		t.Errorf("results[0] = %+v, want zero value with %v", got[0], boom)
	}
	if got[1].Value != 4 || got[1].Err != nil {
		t.Errorf("results[1] = %+v, want {Value:4 Err:<nil>}", got[1])
	}
	if got[2].Value != 6 {
		t.Errorf("results[2] = %+v, want Value 6", got[2])
	}
}

func TestInvalidWorkers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		workers int
	}{
		{name: "negative workers", workers: -1},
		{name: "zero workers", workers: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := canceledContext()
			fn := mustNotCall(t)

			got, err := ParallelMap(ctx, tc.workers, []int{1}, fn)

			if got != nil || !errors.Is(err, ErrInvalidWorkers) {
				t.Fatalf("got %v, %v; want nil, %v", got, err, ErrInvalidWorkers)
			}
		})
	}
}

func TestCanceledContext(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []int
	}{
		{name: "empty input", in: []int{}},
		{name: "non-empty input", in: []int{1, 2, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := canceledContext()
			fn := mustNotCall(t)

			got, err := ParallelMap(ctx, 2, tc.in, fn)

			if got != nil || !errors.Is(err, context.Canceled) {
				t.Fatalf("got %v, %v; want nil, %v", got, err, context.Canceled)
			}
		})
	}
}

func TestEmptyInput(t *testing.T) {
	fn := mustNotCall(t)

	got, err := ParallelMap(context.Background(), 1, []int{}, fn)

	if len(got) != 0 || err != nil {
		t.Fatalf("got %v, %v; want empty, nil", got, err)
	}
}

func TestBoundedOrderAndJoin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			n       = 20
			workers = 3
		)
		in := make([]int, n)
		gates := make([]chan struct{}, n)
		for i := range n {
			in[i] = i
			gates[i] = make(chan struct{})
		}
		var active, peak, calls atomic.Int32
		fn := func(_ context.Context, x int) (int, error) {
			a := active.Add(1)
			for p := peak.Load(); a > p; p = peak.Load() {
				if peak.CompareAndSwap(p, a) {
					break
				}
			}
			calls.Add(1)
			<-gates[x]
			active.Add(-1)
			return x * 2, nil
		}

		var got []Result[int]
		var err error
		done := make(chan struct{})
		go func() {
			defer close(done)
			got, err = ParallelMap(context.Background(), workers, in, fn)
		}()
		synctest.Wait()
		blockedCalls := calls.Load()
		for i := n - 1; i >= 0; i-- {
			close(gates[i])
		}
		<-done

		if blockedCalls != workers {
			t.Fatalf("expected %d blocked workers, got %d", workers, blockedCalls)
		}
		if err != nil || len(got) != n {
			t.Fatalf("got %d results, err %v; want %d, nil", len(got), err, n)
		}
		if peak.Load() > workers || active.Load() != 0 {
			t.Fatalf("peak=%d active=%d; want peak<=%d active=0", peak.Load(), active.Load(), workers)
		}
		for i, r := range got {
			if r.Value != i*2 || r.Err != nil {
				t.Fatalf("results[%d] = %+v, want {Value:%d Err:<nil>}", i, r, i*2)
			}
		}
	})
}

func TestCancelWaitsForWorkers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const workers = 2
		ctx, cancel := context.WithCancel(context.Background())
		var started, ended atomic.Int32
		fn := func(ctx context.Context, _ int) (int, error) {
			started.Add(1)
			<-ctx.Done()
			time.Sleep(time.Second) // cleanup after cancellation; fake time inside synctest
			ended.Add(1)
			return 0, ctx.Err()
		}

		var got []Result[int]
		var err error
		var endedAtReturn int32
		done := make(chan struct{})
		go func() {
			defer close(done)
			got, err = ParallelMap(ctx, workers, []int{1, 2, 3, 4}, fn)
			endedAtReturn = ended.Load()
		}()
		synctest.Wait()
		startedBeforeCancel := started.Load()
		cancel()
		<-done
		// Let workers leaked by a non-waiting solution finish, so the bubble
		// exits cleanly and the failure below is reported without a deadlock panic.
		time.Sleep(2 * time.Second)

		if startedBeforeCancel != workers {
			t.Fatalf("pool did not fill: %d workers started, want %d", startedBeforeCancel, workers)
		}
		if got != nil || !errors.Is(err, context.Canceled) {
			t.Errorf("got %v, %v; want nil, %v", got, err, context.Canceled)
		}
		if endedAtReturn != startedBeforeCancel {
			t.Fatalf("returned before workers finished: started=%d ended=%d", startedBeforeCancel, endedAtReturn)
		}
		if started.Load() != startedBeforeCancel {
			t.Fatalf("dispatch continued after cancel: started=%d, want %d", started.Load(), startedBeforeCancel)
		}
	})
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func mustNotCall(t *testing.T) func(context.Context, int) (int, error) {
	return func(context.Context, int) (int, error) {
		t.Error("fn must not be called")
		return 0, nil
	}
}
