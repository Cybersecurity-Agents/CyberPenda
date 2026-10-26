package runtime

import (
	"context"
	"errors"
	"pentest/internal/task"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRuntimeOutputBatchFlushesAtBoundAndContinuationBoundary(t *testing.T) {
	var sizes []int
	var continuations []string
	b := newRuntimeOutputBatch(func(c string, p []task.EventPayload) error {
		sizes = append(sizes, len(p))
		continuations = append(continuations, c)
		return nil
	}, func(error) {})
	b.delay = time.Hour
	for i := 0; i < 33; i++ {
		b.Append("first", task.EventPayload{"text": "line"})
	}
	b.Append("second", task.EventPayload{"text": "next"})
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sizes, []int{32, 1, 1}) || !reflect.DeepEqual(continuations, []string{"first", "first", "second"}) {
		t.Fatalf("batches=%v continuations=%v", sizes, continuations)
	}
}

func TestRuntimeOutputBatchRetriesAtomicFailureAndReportsIt(t *testing.T) {
	calls := 0
	failure := errors.New("store failed")
	notified := 0
	b := newRuntimeOutputBatch(func(c string, p []task.EventPayload) error {
		calls++
		if calls == 1 {
			return failure
		}
		if len(p) != 1 || p[0]["text"] != "line" {
			t.Fatalf("retry lost output: %+v", p)
		}
		return nil
	}, func(error) { notified++ })
	b.delay = time.Hour
	b.Append("one", task.EventPayload{"text": "line"})
	if err := b.Flush(); !errors.Is(err, failure) {
		t.Fatalf("flush error=%v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("final retry error=%v", err)
	}
	if !errors.Is(b.Err(), failure) {
		t.Fatalf("original failure lost: %v", b.Err())
	}
	if calls != 2 || notified != 1 {
		t.Fatalf("calls=%d notified=%d", calls, notified)
	}
}

func TestRuntimeOutputBatchFlushesIdleOutput(t *testing.T) {
	flushed := make(chan string, 1)
	b := newRuntimeOutputBatch(func(c string, p []task.EventPayload) error { flushed <- p[0]["text"].(string); return nil }, func(error) {})
	defer b.Close()
	b.Append("one", task.EventPayload{"text": "visible"})
	select {
	case text := <-flushed:
		if text != "visible" {
			t.Fatal(text)
		}
	case <-time.After(time.Second):
		t.Fatal("output never became durable")
	}
}

type batchOrderAdapter struct{}

func (batchOrderAdapter) Name() string { return "batch-order" }
func (batchOrderAdapter) Run(_ context.Context, _ string, emit func(task.EventKind, task.EventPayload)) error {
	for i := 0; i < 65; i++ {
		emit(task.EventKindRuntimeOutput, task.EventPayload{"text": strconv.Itoa(i)})
	}
	emit(task.EventKindLifecycle, task.EventPayload{"phase": "provider_permission_requested"})
	emit(task.EventKindRuntimeOutput, task.EventPayload{"text": "65"})
	return nil
}

func TestOwnerHarnessFlushesOutputBeforePermissionAndFinalization(t *testing.T) {
	var mu sync.Mutex
	var order []string
	batches := 0
	h := NewOwnerHarness(OwnerHarnessConfig{
		VerifyOwner: func(string) error { return nil },
		AppendRuntimeOutputs: func(owner, c string, payloads []task.EventPayload) error {
			mu.Lock()
			defer mu.Unlock()
			if len(payloads) > runtimeOutputBatchLimit {
				t.Fatalf("unbounded batch: %d", len(payloads))
			}
			batches++
			if owner != "one" || c != "continuation" {
				t.Fatalf("wrong owner: %s %s", owner, c)
			}
			for _, p := range payloads {
				order = append(order, p["text"].(string))
			}
			return nil
		},
		AppendEvent: func(_, _ string, _ task.EventKind, p task.EventPayload) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, p["phase"].(string))
			return nil
		},
		Finalize: func(_, _, status string) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, "finalize:"+status)
			return nil
		},
	})
	if err := h.Launch(context.Background(), OwnerLaunchRequest{OwnerID: "one", ContinuationID: "continuation", Adapter: batchOrderAdapter{}}); err != nil {
		t.Fatal(err)
	}
	want := []string{"started"}
	for i := 0; i < 65; i++ {
		want = append(want, strconv.Itoa(i))
	}
	want = append(want, "provider_permission_requested", "65", "completed", "finalize:completed")
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("durable event order=%v, want %v", order, want)
	}
	if batches == 0 {
		t.Fatal("output batch callback was not used")
	}
}

func TestOwnerHarnessReportsFailureAndRejectsLaterOutputAfterCancellation(t *testing.T) {
	failure := errors.New("temporary atomic append failure")
	calls := 0
	var mu sync.Mutex
	var output []string
	terminalAt := -1
	h := NewOwnerHarness(OwnerHarnessConfig{
		VerifyOwner: func(string) error { return nil },
		AppendRuntimeOutputs: func(_, _ string, payloads []task.EventPayload) error {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 1 {
				return failure
			}
			for _, p := range payloads {
				output = append(output, p["text"].(string))
			}
			return nil
		},
		AppendEvent: func(_, _ string, _ task.EventKind, p task.EventPayload) error {
			mu.Lock()
			defer mu.Unlock()
			if p["phase"] == "stopped" {
				terminalAt = len(output)
			}
			return nil
		},
	})
	err := h.Launch(context.Background(), OwnerLaunchRequest{OwnerID: "one", ContinuationID: "continuation", Adapter: batchOrderAdapter{}})
	if !errors.Is(err, failure) {
		t.Fatalf("original store error lost: %v", err)
	}
	if len(output) != 32 || terminalAt != 32 {
		t.Fatalf("outputs=%d terminal at=%d", len(output), terminalAt)
	}
	for i, text := range output {
		if text != strconv.Itoa(i) {
			t.Fatalf("row %d = %q", i, text)
		}
	}
}

func TestRuntimeOutputBatchRetainsOnlyBoundedFailedBatch(t *testing.T) {
	failure := errors.New("write failed")
	b := newRuntimeOutputBatch(func(string, []task.EventPayload) error { return failure }, func(error) {})
	b.delay = time.Hour
	for i := 0; i < 10000; i++ {
		b.Append("one", task.EventPayload{"text": "line"})
	}
	if len(b.pending) != runtimeOutputBatchLimit || b.rejected != 10000-runtimeOutputBatchLimit {
		t.Fatalf("pending=%d rejected=%d", len(b.pending), b.rejected)
	}
	if !errors.Is(b.Close(), failure) || !errors.Is(b.Err(), failure) {
		t.Fatal("failure was not propagated")
	}
}
