package runtime_test

import (
	"context"
	"pentest/internal/runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostBridgeRegistryCreatesOwnersIndependently(t *testing.T) {
	registry := runtime.NewHostSessionBridgeRegistry()
	defer registry.CloseAll(context.Background())
	testRegistryOwnerCreation(t, func(id string, create func()) error {
		_, err := registry.Bind(context.Background(), id, id+"-continuation", func() (*runtime.HostSessionBridge, error) {
			create()
			return runtime.NewHostSessionBridge(runtime.HostSessionBridgeConfig{TaskID: id, Program: "unused"})
		})
		return err
	})
}

func TestSandboxBridgeRegistryCreatesOwnersIndependently(t *testing.T) {
	registry := runtime.NewSandboxSessionBridgeRegistry()
	defer registry.CloseAll(context.Background())
	testRegistryOwnerCreation(t, func(id string, create func()) error {
		_, err := registry.Bind(context.Background(), id, id+"-continuation", func() (*runtime.SandboxSessionBridge, error) {
			create()
			return runtime.NewSandboxSessionBridge(newFakeBridgeDocker(), runtime.SandboxBridgeConfig{TaskID: id, CreateArgs: []string{"create", "-i", "image", "bridge"}})
		})
		return err
	})
}

func testRegistryOwnerCreation(t *testing.T, bind func(string, func()) error) {
	t.Helper()
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var creates atomic.Int32
	go func() { done <- bind("held", func() { creates.Add(1); close(entered); <-release }) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first create did not enter")
	}
	other := make(chan error, 1)
	go func() { other <- bind("other", func() {}) }()
	select {
	case err := <-other:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(time.Second):
		t.Error("unrelated owner waited for create")
	}
	same := make(chan error, 1)
	go func() { same <- bind("held", func() { creates.Add(1) }) }()
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-same; err != nil {
		t.Fatal(err)
	}
	if creates.Load() != 1 {
		t.Fatalf("same owner created %d bridges", creates.Load())
	}
}

func TestHostBridgeRegistryCloseWaitsForOwnerCreation(t *testing.T) {
	registry := runtime.NewHostSessionBridgeRegistry()
	entered, release := make(chan struct{}), make(chan struct{})
	bound := make(chan *runtime.HostSessionBridge, 1)
	bindErr := make(chan error, 1)
	go func() {
		bridge, err := registry.Bind(context.Background(), "held", "c1", func() (*runtime.HostSessionBridge, error) {
			close(entered)
			<-release
			return runtime.NewHostSessionBridge(runtime.HostSessionBridgeConfig{TaskID: "held", Program: "unused"})
		})
		bound <- bridge
		bindErr <- err
	}()
	<-entered
	closed := make(chan error, 1)
	go func() { closed <- registry.CloseTask(context.Background(), "held") }()
	close(release)
	bridge := <-bound
	if err := <-bindErr; err != nil {
		t.Fatal(err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get("held"); ok {
		t.Fatal("Close retained owner")
	}
	select {
	case <-bridge.Closed():
	default:
		t.Fatal("Close did not close bridge created concurrently")
	}
}
