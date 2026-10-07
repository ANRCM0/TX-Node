package reporting

import (
	"sync"
	"sync/atomic"

	"github.com/ANRCM0/TX-Node/internal/controlplane"
)

// Batch is one immutable report submission prepared by the Service adapter.
// The payload contains the already-flushed runtime counters that may need to be
// restored when an asynchronous delivery fails.
type Batch struct {
	Payload      controlplane.ReportPayload
	TrafficCount int
	OnlineCount  int
}

type PrepareFunc func() Batch
type FailureFunc func(Batch, error)
type SuccessFunc func(Batch)

// Controller owns report-delivery mechanics only: overlap prevention, bounded
// retry backoff and asynchronous/synchronous Sink.Report invocation.
//
// Runtime metric collection and tracker flush/restore remain behind Service
// callbacks so this package cannot mutate unrelated data-plane state.
type Controller struct {
	sink controlplane.Sink

	active  atomic.Bool
	backoff backoff
}

func New(sink controlplane.Sink) *Controller {
	return &Controller{sink: sink}
}

func (c *Controller) SupportsReporting() bool {
	return c != nil && c.sink != nil && c.sink.SupportsReporting()
}

// PushAsync starts one report if reporting is supported, no previous report is
// active and retry backoff permits the attempt. Prepare is invoked only after
// those checks, preserving the old rule that skipped reports do not flush
// tracker state.
func (c *Controller) PushAsync(
	prepare PrepareFunc,
	onFailure FailureFunc,
	onSuccess SuccessFunc,
) bool {
	if !c.SupportsReporting() || prepare == nil {
		return false
	}
	if !c.active.CompareAndSwap(false, true) {
		return false
	}
	if c.backoff.shouldSkip() {
		c.active.Store(false)
		return false
	}

	batch := prepare()
	go func() {
		defer c.active.Store(false)

		if err := c.sink.Report(batch.Payload); err != nil {
			c.backoff.onFailure()
			if onFailure != nil {
				onFailure(batch, err)
			}
			return
		}

		c.backoff.onSuccess()
		if onSuccess != nil {
			onSuccess(batch)
		}
	}()
	return true
}

// PushSync is used for the shutdown flush. It intentionally bypasses async
// overlap/backoff state, matching the previous Service behavior.
func (c *Controller) PushSync(prepare PrepareFunc) error {
	if !c.SupportsReporting() || prepare == nil {
		return nil
	}
	batch := prepare()
	return c.sink.Report(batch.Payload)
}

func (c *Controller) Active() bool {
	return c != nil && c.active.Load()
}

type backoff struct {
	mu            sync.Mutex
	skipRemaining int
}

func (b *backoff) shouldSkip() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.skipRemaining > 0 {
		b.skipRemaining--
		return true
	}
	return false
}

func (b *backoff) onSuccess() {
	b.mu.Lock()
	b.skipRemaining = 0
	b.mu.Unlock()
}

func (b *backoff) onFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.skipRemaining <= 0 {
		b.skipRemaining = 1
	} else if b.skipRemaining < 8 {
		b.skipRemaining *= 2
	}
}
