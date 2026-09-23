package nodesync

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/PaiMonCai/TX-Node/internal/controlplane"
	"github.com/PaiMonCai/TX-Node/internal/model"
)

type fakeSource struct {
	mu           sync.Mutex
	polls        int
	pollResult   controlplane.Snapshot
	pollErr      error
	pollStarted  chan struct{}
	pollRelease  chan struct{}
	supportsPoll bool
}

func (f *fakeSource) Initial(
	context.Context,
	func() map[string]interface{},
	chan<- controlplane.Event,
	chan<- controlplane.StatusChange,
) (controlplane.Bootstrap, error) {
	return controlplane.Bootstrap{}, nil
}

func (f *fakeSource) Poll(ctx context.Context) (controlplane.Snapshot, error) {
	f.mu.Lock()
	f.polls++
	started := f.pollStarted
	release := f.pollRelease
	result := f.pollResult
	err := f.pollErr
	f.mu.Unlock()

	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return controlplane.Snapshot{}, ctx.Err()
		}
	}
	return result, err
}

func (f *fakeSource) Discover(
	context.Context,
	func() map[string]interface{},
	chan<- controlplane.Event,
	chan<- controlplane.StatusChange,
) (controlplane.PushClient, error) {
	return nil, nil
}

func (f *fakeSource) Metrics() controlplane.APIMetrics { return controlplane.APIMetrics{} }
func (f *fakeSource) SupportsPolling() bool            { return f.supportsPoll }
func (f *fakeSource) SupportsDiscovery() bool          { return false }

func (f *fakeSource) pollCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.polls
}

func TestControllerPollPreparesHashedSnapshot(t *testing.T) {
	source := &fakeSource{
		supportsPoll: true,
		pollResult: controlplane.Snapshot{
			Config: &model.NodeSpec{Protocol: "vless", ServerPort: 443},
			Users: []model.UserSpec{
				{ID: 2, UUID: "b", SpeedLimit: 20},
				{ID: 1, UUID: "a", SpeedLimit: 10},
			},
		},
	}
	controller := New(source)

	if !controller.Poll(context.Background(), "", false) {
		t.Fatal("expected poll to start")
	}

	select {
	case result := <-controller.Results():
		if result.Config == nil || result.ConfigHash == "" {
			t.Fatalf("missing config result: %#v", result)
		}
		if len(result.Users) != 2 || result.UserHash == "" {
			t.Fatalf("missing user result: %#v", result)
		}
		if result.UserHash != UserHash([]model.UserSpec{
			{ID: 1, UUID: "a", SpeedLimit: 10},
			{ID: 2, UUID: "b", SpeedLimit: 20},
		}) {
			t.Fatal("user hash must be independent of source ordering")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for poll result")
	}
}

func TestControllerSuppressesUnchangedConfigUnlessCertificateChanged(t *testing.T) {
	cfg := &model.NodeSpec{Protocol: "vless", ServerPort: 443}
	source := &fakeSource{
		supportsPoll: true,
		pollResult:   controlplane.Snapshot{Config: cfg},
	}
	controller := New(source)
	hash := ConfigHash(cfg)

	controller.Poll(context.Background(), hash, false)
	result := <-controller.Results()
	if result.Config != nil {
		t.Fatalf("unchanged config should be suppressed: %#v", result.Config)
	}

	controller.Poll(context.Background(), hash, true)
	result = <-controller.Results()
	if result.Config == nil || !result.CertChanged {
		t.Fatalf("certificate renewal must force config application: %#v", result)
	}
}

func TestControllerPreventsOverlappingPolls(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	source := &fakeSource{
		supportsPoll: true,
		pollStarted:  started,
		pollRelease:  release,
	}
	controller := New(source)

	if !controller.Poll(context.Background(), "", false) {
		t.Fatal("first poll should start")
	}
	<-started
	if controller.Poll(context.Background(), "", false) {
		t.Fatal("overlapping poll must be rejected")
	}
	if got := source.pollCount(); got != 1 {
		t.Fatalf("poll count = %d, want 1", got)
	}
	close(release)
	<-controller.Results()
}

func TestControllerBacksOffAfterFailure(t *testing.T) {
	source := &fakeSource{
		supportsPoll: true,
		pollErr:      errors.New("panel unavailable"),
	}
	controller := New(source)

	if !controller.Poll(context.Background(), "", false) {
		t.Fatal("first poll should start")
	}
	deadline := time.Now().Add(time.Second)
	for source.pollCount() < 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(10 * time.Millisecond)

	if controller.Poll(context.Background(), "", false) {
		t.Fatal("first retry should be skipped by backoff")
	}

	source.mu.Lock()
	source.pollErr = nil
	source.mu.Unlock()

	if !controller.Poll(context.Background(), "", false) {
		t.Fatal("poll after consumed backoff should start")
	}
	select {
	case <-controller.Results():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for successful retry")
	}
}

func TestControllerCoalescesResyncRequests(t *testing.T) {
	source := &fakeSource{
		supportsPoll: true,
		pollResult:   controlplane.Snapshot{},
	}
	controller := New(source)

	if !controller.RequestResync(context.Background(), "", false) {
		t.Fatal("first resync request should be accepted")
	}
	if controller.RequestResync(context.Background(), "", false) {
		t.Fatal("duplicate resync request must be coalesced")
	}
	<-controller.Results()

	controller.CompleteResync()
	if !controller.RequestResync(context.Background(), "", false) {
		t.Fatal("resync should be accepted after completion")
	}
	<-controller.Results()
}
