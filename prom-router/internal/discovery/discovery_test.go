package discovery

import (
	"testing"
)

type dummyService struct {
	Value string
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	// Test Register and Get success
	expectedService := &dummyService{Value: "test_val"}
	reg.Register("DummyService", expectedService)

	svc, err := reg.Get("DummyService")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	dummy, ok := svc.(*dummyService)
	if !ok {
		t.Fatalf("expected type *dummyService, got %T", svc)
	}
	if dummy.Value != "test_val" {
		t.Errorf("expected value 'test_val', got '%s'", dummy.Value)
	}

	// Test Get not found
	_, err = reg.Get("NonExistentService")
	if err == nil {
		t.Fatalf("expected error for non-existent service, got nil")
	}
}
