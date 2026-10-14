package collector

import "testing"

func TestRegistryTypes(t *testing.T) {
	r := NewRegistry()
	if len(r.Types()) != 0 {
		t.Fatal("expected empty registry")
	}
}
