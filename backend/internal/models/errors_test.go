package models

import (
	"errors"
	"testing"
)

func TestWrapInternal(t *testing.T) {
	if WrapInternal(nil) != nil {
		t.Fatal("WrapInternal(nil) want nil")
	}

	orig := Invalid("bad")
	got := WrapInternal(orig)
	if got != orig {
		t.Fatalf("WrapInternal(api error) = %p, want same pointer", got)
	}

	wrapped := WrapInternal(errors.New("boom"))
	if wrapped == nil || wrapped.Kind != KindInternal || wrapped.Message != "boom" {
		t.Fatalf("WrapInternal(plain) = %+v", wrapped)
	}
}

func TestWrapInternalPreservesKinds(t *testing.T) {
	for _, orig := range []*Error{
		Invalid("bad"),
		NotFound("gone"),
		Conflict("exists"),
		ConflictState("wrong phase"),
		Internal("boom"),
	} {
		got := WrapInternal(orig)
		if got != orig {
			t.Fatalf("WrapInternal(%s) = %+v, want same pointer", orig.Kind, got)
		}
	}
}

func TestAsError(t *testing.T) {
	var target *Error
	if AsError(nil, &target) {
		t.Fatal("AsError(nil) want false")
	}
	if AsError(errors.New("nope"), &target) {
		t.Fatal("AsError(plain) want false")
	}
	src := NotFound("gone")
	if !AsError(src, &target) || target != src {
		t.Fatalf("AsError(api) = %v %v", target, src)
	}
}
