package runtimeplugin_test

import (
	"pentest/internal/runtimeplugin"
	"reflect"
	"sync"
	"testing"
)

func TestBuiltinRegistrySharedAndIsolated(t *testing.T) {
	first := runtimeplugin.MustBuiltinRegistry()
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if got := runtimeplugin.MustBuiltinRegistry(); got != first {
				t.Error("builtin registry was rebuilt")
			}
		}()
	}
	workers.Wait()
	baseline, ok := first.Get("codex")
	if !ok {
		t.Fatal("missing codex")
	}
	changed, _ := first.Get("codex")
	if len(changed.ConfigProjection.ManagedKeys) == 0 || len(changed.NativeResume.Args) == 0 {
		t.Fatal("missing mutation test inputs")
	}
	changed.ConfigProjection.ManagedKeys[0].Key = "changed"
	changed.NativeResume.Args[0] = "changed"
	again, _ := first.Get("codex")
	if !reflect.DeepEqual(again, baseline) {
		t.Fatal("Get exposes shared plugin state")
	}
	listed := first.List()
	for i := range listed {
		if listed[i].ID == "codex" {
			listed[i].ConfigProjection.ManagedKeys[0].Key = "list changed"
			listed[i].NativeResume.Args[0] = "list changed"
		}
	}
	again, _ = first.Get("codex")
	if !reflect.DeepEqual(again, baseline) {
		t.Fatal("List exposes shared plugin state")
	}
}

func BenchmarkBuiltinRegistryCached(b *testing.B) {
	runtimeplugin.MustBuiltinRegistry()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		runtimeplugin.MustBuiltinRegistry()
	}
}
