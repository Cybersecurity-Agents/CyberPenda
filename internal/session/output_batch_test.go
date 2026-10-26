package session

import (
	"encoding/json"
	"path/filepath"
	"pentest/internal/store"
	"testing"
)

func TestAppendRuntimeOutputsPreservesRowsAndRollsBackFailedBatch(t *testing.T) {
	root := t.TempDir()
	db, err := store.Open(filepath.Join(root, "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewService(db, filepath.Join(root, "sessions"))
	created, err := s.Create(CreateRequest{Input: "Inspect"})
	if err != nil {
		t.Fatal(err)
	}
	inputs := []EventPayload{{"text": "one", "stream": "stdout"}, {"text": "two", "stream": "stderr"}}
	events, err := s.AppendRuntimeOutputs(created.ID, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Seq != 2 || events[1].Seq != 3 {
		t.Fatalf("batch rows: %+v", events)
	}
	stored, err := s.Events(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range inputs {
		a, _ := json.Marshal(p)
		b, _ := json.Marshal(stored[i+1].Payload)
		if string(a) != string(b) {
			t.Fatalf("payload changed: %s vs %s", a, b)
		}
	}
	_, err = s.AppendRuntimeOutputs(created.ID, []EventPayload{{"text": "must roll back"}, {"invalid": make(chan int)}})
	if err == nil {
		t.Fatal("unsupported payload should fail batch")
	}
	stored, err = s.Events(created.ID)
	if err != nil || len(stored) != 3 {
		t.Fatalf("partial batch committed: %+v %v", stored, err)
	}
}
