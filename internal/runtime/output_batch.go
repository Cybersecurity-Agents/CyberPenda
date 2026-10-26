package runtime

import (
	"fmt"
	"sync"
	"time"

	"pentest/internal/task"
)

const runtimeOutputBatchLimit = 32
const runtimeOutputBatchBytes = 1 << 20
const runtimeOutputBatchDelay = 10 * time.Millisecond

type queuedRuntimeOutput struct {
	continuation string
	payload      task.EventPayload
}
type runtimeOutputBatch struct {
	mu         sync.Mutex
	appendRows func(string, []task.EventPayload) error
	failed     func(error)
	pending    []queuedRuntimeOutput
	timer      *time.Timer
	closed     bool
	closeError error
	rejected   int
	firstError error
	delay      time.Duration
}

func newRuntimeOutputBatch(appendRows func(string, []task.EventPayload) error, failed func(error)) *runtimeOutputBatch {
	return &runtimeOutputBatch{appendRows: appendRows, failed: failed, delay: runtimeOutputBatchDelay}
}

func (b *runtimeOutputBatch) Append(continuation string, payload task.EventPayload) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	if b.firstError != nil {
		b.rejected++
		return
	}
	size := 0
	for _, p := range b.pending {
		text, _ := p.payload["text"].(string)
		size += len(text)
	}
	text, _ := payload["text"].(string)
	if len(b.pending) > 0 && (b.pending[0].continuation != continuation || size+len(text) > runtimeOutputBatchBytes) {
		if err := b.flushLocked(); err != nil {
			b.rejected++
			return
		}
		size = 0
	}

	copyPayload := make(task.EventPayload, len(payload))
	for k, v := range payload {
		copyPayload[k] = v
	}
	b.pending = append(b.pending, queuedRuntimeOutput{continuation, copyPayload})
	if len(b.pending) >= runtimeOutputBatchLimit || size+len(text) >= runtimeOutputBatchBytes {
		_ = b.flushLocked()
	}

	if len(b.pending) > 0 && b.timer == nil && b.firstError == nil {
		b.timer = time.AfterFunc(b.delay, func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			b.timer = nil
			if !b.closed {
				_ = b.flushLocked()
			}
		})
	}
}

func (b *runtimeOutputBatch) flushLocked() error {
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	for len(b.pending) > 0 {
		continuation := b.pending[0].continuation
		payloads := make([]task.EventPayload, 0, runtimeOutputBatchLimit)
		size := 0
		for _, p := range b.pending {
			text, _ := p.payload["text"].(string)
			if len(payloads) > 0 && (p.continuation != continuation || len(payloads) >= runtimeOutputBatchLimit || size+len(text) > runtimeOutputBatchBytes) {
				break
			}
			payloads = append(payloads, p.payload)
			size += len(text)
		}
		if err := b.appendRows(continuation, payloads); err != nil {
			if b.firstError == nil {
				b.firstError = err
				b.failed(err)
			}
			return err // The atomic append failed. Keep the entire batch for final retry.
		}
		for i := range payloads {
			b.pending[i] = queuedRuntimeOutput{}
		}
		b.pending = b.pending[len(payloads):]
	}
	return nil
}

func (b *runtimeOutputBatch) Flush() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return b.closeError
	}
	return b.flushLocked()
}
func (b *runtimeOutputBatch) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return b.closeError
	}
	b.closeError = b.flushLocked()
	b.closed = true
	return b.closeError
}
func (b *runtimeOutputBatch) Err() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.firstError != nil && b.rejected > 0 {
		return fmt.Errorf("Runtime output append failed; rejected %d later events after cancellation: %w", b.rejected, b.firstError)
	}
	return b.firstError
}
