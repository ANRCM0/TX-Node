package reporting

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

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

	pendingMu sync.Mutex
	pending   *Batch
}

var fallbackNonce atomic.Uint64

func newReportBatchID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// Only used if OS entropy fails; still distinct across calls/processes.
	return fmt.Sprintf("fallback-%x-%x-%x", time.Now().UnixNano(), os.Getpid(), fallbackNonce.Add(1))
}

// Pending traffic is NOT returned to the tracker after an ambiguous HTTP
// failure. Replaying a byte-identical payload with the same ID is the only
// safe way to distinguish a lost response from a failed commit.
func (c *Controller) nextBatch(prepare PrepareFunc) Batch {
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	if c.pending != nil {
		return *c.pending
	}

	batch := prepare()
	if len(batch.Payload.Traffic) > 0 {
		batch.Payload.BatchID = newReportBatchID()
		snapshot := batch
		c.pending = &snapshot
	}
	return batch
}

func (c *Controller) acknowledge(batch Batch) {
	if batch.Payload.BatchID == "" {
		return
	}
	c.pendingMu.Lock()
	defer c.pendingMu.Unlock()
	if c.pending != nil && c.pending.Payload.BatchID == batch.Payload.BatchID {
		c.pending = nil
	}
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

	batch := c.nextBatch(prepare)
	go func() {
		defer c.active.Store(false)

		if err := c.sink.Report(batch.Payload); err != nil {
			c.backoff.onFailure()
			if onFailure != nil {
				// The controller retains these counters for a stable replay.
				// Only transient alive IP snapshots may return to the tracker.
				failureNotice := batch
				if batch.Payload.BatchID != "" {
					failureNotice.Payload.Traffic = nil
				}
				onFailure(failureNotice, err)
			}
			return
		}

		c.acknowledge(batch)
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
	// On shutdown, attempt the ambiguous in-flight batch first and only
	// then flush new counters; do not roll them into the old batch identity.
	c.pendingMu.Lock()
	hadPending := c.pending != nil
	c.pendingMu.Unlock()
	batch := c.nextBatch(prepare)
	if err := c.sink.Report(batch.Payload); err != nil {
		return err
	}
	c.acknowledge(batch)
	if hadPending {
		next := c.nextBatch(prepare)
		if err := c.sink.Report(next.Payload); err != nil {
			return err
		}
		c.acknowledge(next)
	}
	return nil
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
