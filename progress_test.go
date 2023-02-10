package mpb_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

const (
	timeout = 300 * time.Millisecond
)

func TestWithContext(t *testing.T) {
	shutdown := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	p := mpb.NewWithContext(ctx,
		mpb.WithShutdownNotifier(shutdown),
	)
	_ = p.AddBar(0) // never complete bar
	_ = p.AddBar(0) // never complete bar
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	p.Wait()

	select {
	case v := <-shutdown:
		if l := len(v.([]*mpb.Bar)); l != 2 {
			t.Errorf("Expected len of bars: %d, got: %d", 2, l)
		}
	case <-time.After(timeout):
		t.Errorf("Progress didn't shutdown after %v", timeout)
	}
}

func TestShutdownsWithErrFiller(t *testing.T) {
	var debug bytes.Buffer
	shutdown := make(chan interface{})
	p := mpb.New(
		mpb.WithShutdownNotifier(shutdown),
		mpb.WithOutput(io.Discard),
		mpb.WithDebugOutput(&debug),
		mpb.ForceAutoRefresh(),
	)

	var errReturnCount int
	testError := errors.New("test error")
	bar := p.AddBar(100,
		mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
			return mpb.BarFillerFunc(func(w io.Writer, st decor.Statistics) error {
				if st.Current >= 22 {
					errReturnCount++
					return testError
				}
				return base.Fill(w, st)
			})
		}),
	)

	go func() {
		for bar.IsRunning() {
			bar.Increment()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	p.Wait()

	if errReturnCount != 1 {
		t.Errorf("Expected errReturnCount: %d, got: %d\n", 1, errReturnCount)
	}

	select {
	case v := <-shutdown:
		if l := len(v.([]*mpb.Bar)); l != 0 {
			t.Errorf("Expected len of bars: %d, got: %d\n", 0, l)
		}
		if err := strings.TrimSpace(debug.String()); err != testError.Error() {
			t.Errorf("Expected err: %q, got %q\n", testError.Error(), err)
		}
	case <-time.After(timeout):
		t.Errorf("Progress didn't shutdown after %v", timeout)
	}
}

func TestShutdownAfterBarAbortWithDrop(t *testing.T) {
	shutdown := make(chan interface{})
	p := mpb.New(
		mpb.WithShutdownNotifier(shutdown),
		mpb.WithOutput(io.Discard),
		mpb.ForceAutoRefresh(),
	)
	b := p.AddBar(100)

	var count int
	for i := 0; !b.Aborted(); i++ {
		if i >= 10 {
			count++
			b.Abort(true)
		} else {
			b.Increment()
			time.Sleep(10 * time.Millisecond)
		}
	}

	p.Wait()

	if count != 1 {
		t.Errorf("Expected count: %d, got: %d", 1, count)
	}

	select {
	case v := <-shutdown:
		if l := len(v.([]*mpb.Bar)); l != 0 {
			t.Errorf("Expected len of bars: %d, got: %d", 0, l)
		}
	case <-time.After(timeout):
		t.Errorf("Progress didn't shutdown after %v", timeout)
	}
}

func TestShutdownAfterBarAbortWithNoDrop(t *testing.T) {
	shutdown := make(chan interface{})
	p := mpb.New(
		mpb.WithShutdownNotifier(shutdown),
		mpb.WithOutput(io.Discard),
		mpb.ForceAutoRefresh(),
	)
	b := p.AddBar(100)

	var count int
	for i := 0; !b.Aborted(); i++ {
		if i >= 10 {
			count++
			b.Abort(false)
		} else {
			b.Increment()
			time.Sleep(10 * time.Millisecond)
		}
	}

	p.Wait()

	if count != 1 {
		t.Errorf("Expected count: %d, got: %d", 1, count)
	}

	select {
	case v := <-shutdown:
		if l := len(v.([]*mpb.Bar)); l != 1 {
			t.Errorf("Expected len of bars: %d, got: %d", 1, l)
		}
	case <-time.After(timeout):
		t.Errorf("Progress didn't shutdown after %v", timeout)
	}
}
