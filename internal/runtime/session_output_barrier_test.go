package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"pentest/internal/session"
	"pentest/internal/store"
	"pentest/internal/task"
)

type outputBarrierAdapter struct {
	emitted chan struct{}
	release chan struct{}
	payload task.EventPayload
}

func (outputBarrierAdapter) Name() string { return "output-barrier" }
func (a outputBarrierAdapter) Run(ctx context.Context, _ string, emit func(task.EventKind, task.EventPayload)) error {
	emit(task.EventKindRuntimeOutput, a.payload)
	close(a.emitted)
	select {
	case <-a.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type outputBarrierFixture struct {
	db           *store.DB
	sessions     *session.Service
	owner        session.Session
	continuation session.Continuation
	harness      *SessionHarness
	prepared     *session.PreparedConversationInput
}

func newOutputBarrierFixture(t *testing.T) *outputBarrierFixture {
	t.Helper()
	root := t.TempDir()
	db, err := store.Open(filepath.Join(root, "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	sessions := session.NewService(db, filepath.Join(root, "sessions"))
	created, err := sessions.Create(session.CreateRequest{Input: "Inspect"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := sessions.CreateContinuation(created.ID, "profile:one", "codex", session.RunnerHost)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := sessions.PrepareConversationInput(created.ID, c.ID, "user", "B", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(prepared.Rollback)
	h := NewSessionHarness(sessions)
	h.core.config.outputBatchDelay = time.Hour
	emitted, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- h.Launch(context.Background(), SessionLaunchRequest{SessionID: created.ID, ContinuationID: c.ID, Adapter: outputBarrierAdapter{emitted: emitted, release: release, payload: task.EventPayload{"text": "A", "stream": "stdout"}}})
	}()
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("Harness did not release after output barrier test")
		}
	})
	select {
	case <-emitted:
	case <-time.After(5 * time.Second):
		t.Fatal("Runtime did not emit pending output")
	}
	current, err := sessions.Continuation(c.ID)
	if err != nil || current.Status != session.RuntimeStatusRunning || !h.IsActive(created.ID) {
		t.Fatalf("Runtime not durably running: %+v %v", current, err)
	}
	return &outputBarrierFixture{db: db, sessions: sessions, owner: created, continuation: c, harness: h, prepared: prepared}
}

func TestSessionDirectWritersFlushEarlierOutputBeforeCommit(t *testing.T) {
	operations := map[string]func(*outputBarrierFixture) error{
		"event": func(f *outputBarrierFixture) error {
			_, err := f.sessions.AppendEvent(f.owner.ID, session.EventKindConversation, session.EventPayload{"role": "user", "text": "B"})
			return err
		},
		"conversation": func(f *outputBarrierFixture) error {
			_, _, err := f.sessions.AppendConversationInput(f.owner.ID, f.continuation.ID, "user", "B", nil)
			return err
		},
		"provider event": func(f *outputBarrierFixture) error {
			_, err := f.sessions.AppendProviderEvent(f.owner.ID, session.EventKindConversation, session.EventPayload{"role": "user", "text": "B"})
			return err
		},
		"rename": func(f *outputBarrierFixture) error { _, err := f.sessions.Rename(f.owner.ID, "B"); return err },
		"replacement with input": func(f *outputBarrierFixture) error {
			next, err := f.sessions.CreateReplacementContinuationWithInput(f.continuation, nil, session.ConversationInput{Role: "user", Text: "B"})
			if err != nil {
				return err
			}
			return f.harness.RebindContinuation(f.owner.ID, next.ID)
		},
		"prepared caller transaction": func(f *outputBarrierFixture) error {
			// Preparation predates A. The transaction-entry barrier must still flush A.
			if err := f.sessions.FlushRuntimeOutputs(f.owner.ID); err != nil {
				return err
			}
			tx, err := f.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()
			if _, _, err = f.prepared.AppendTx(tx); err != nil {
				return err
			}
			if err = tx.Commit(); err != nil {
				return err
			}
			f.prepared.Commit()
			return nil
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			f := newOutputBarrierFixture(t)
			if err := operation(f); err != nil {
				t.Fatal(err)
			}
			events, err := f.sessions.Events(f.owner.ID)
			if err != nil {
				t.Fatal(err)
			}
			a, b := 0, 0
			for _, e := range events {
				if e.Payload["text"] == "A" {
					a = e.Seq
				}
				if e.Payload["text"] == "B" || e.Payload["title"] == "B" {
					b = e.Seq
				}
			}
			if a == 0 || b <= a {
				t.Fatalf("earlier output out of order: A=%d B=%d events=%+v", a, b, events)
			}
		})
	}
}

func TestSessionTerminalTransitionFlushesPendingOutput(t *testing.T) {
	f := newOutputBarrierFixture(t)
	if _, err := f.sessions.UpdateContinuationStatus(f.continuation.ID, session.RuntimeStatusCompleted); err != nil {
		t.Fatal(err)
	}
	events, err := f.sessions.Events(f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Payload["text"] == "A" {
			found = true
		}
	}
	if !found {
		t.Fatal("Continuation completed before earlier output became durable")
	}
}

func TestSessionFlushFailureRejectsDirectEventCommit(t *testing.T) {
	f := newOutputBarrierFixture(t)
	failure := errors.New("output store failed")
	cleanup := f.sessions.RegisterRuntimeOutputFlush(f.owner.ID, func() error { return failure })
	defer cleanup()
	if _, err := f.sessions.AppendProviderEvent(f.owner.ID, session.EventKindConversation, session.EventPayload{"text": "B"}); !errors.Is(err, failure) {
		t.Fatalf("write accepted after flush failure: %v", err)
	}
	events, err := f.sessions.Events(f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Payload["text"] == "B" {
			t.Fatal("event committed after failed barrier")
		}
	}
}

func TestFailedOutputFlushStillFinalizesSessionContinuation(t *testing.T) {
	root := t.TempDir()
	db, err := store.Open(filepath.Join(root, "failure.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := session.NewService(db, filepath.Join(root, "sessions"))
	owner, err := s.Create(session.CreateRequest{Input: "Inspect"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.CreateContinuation(owner.ID, "profile:one", "codex", session.RunnerHost)
	if err != nil {
		t.Fatal(err)
	}
	h := NewSessionHarness(s)
	emitted, release := make(chan struct{}), make(chan struct{})
	close(release)
	err = h.Launch(context.Background(), SessionLaunchRequest{SessionID: owner.ID, ContinuationID: c.ID, Adapter: outputBarrierAdapter{emitted: emitted, release: release, payload: task.EventPayload{"invalid": make(chan int)}}})
	if err == nil {
		t.Fatal("output failure was hidden")
	}
	current, readErr := s.Continuation(c.ID)
	if readErr != nil || current.Status != session.RuntimeStatusFailed {
		t.Fatalf("failed output left Continuation active: %+v %v", current, readErr)
	}
	if h.IsActive(owner.ID) {
		t.Fatal("failed Harness still active")
	}
	if _, err = s.CreateContinuation(owner.ID, "profile:next", "codex", session.RunnerHost); err != nil {
		t.Fatalf("replacement blocked after failed flush: %v", err)
	}
}
