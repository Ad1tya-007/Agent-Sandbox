package kubernetes

import (
	"errors"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestTranslate(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if Translate(nil) != nil {
			t.Fatal("Translate(nil) want nil")
		}
	})

	t.Run("object not found", func(t *testing.T) {
		err := Translate(apierrors.NewNotFound(schema.GroupResource{Group: SandboxGroup, Resource: SandboxResource}, "demo"))
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
		if errors.Is(err, ErrCRDMissing) {
			t.Fatal("named 404 should not be CRD missing")
		}
	})

	t.Run("already exists", func(t *testing.T) {
		err := Translate(apierrors.NewAlreadyExists(schema.GroupResource{Group: SandboxGroup, Resource: SandboxResource}, "demo"))
		if !errors.Is(err, ErrAlreadyExists) {
			t.Fatalf("got %v, want ErrAlreadyExists", err)
		}
	})

	t.Run("crd missing", func(t *testing.T) {
		err := Translate(&apierrors.StatusError{ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "the server could not find the requested resource",
			Reason:  metav1.StatusReasonNotFound,
			Code:    404,
		}})
		if !errors.Is(err, ErrCRDMissing) {
			t.Fatalf("got %v, want ErrCRDMissing", err)
		}
		if !strings.Contains(err.Error(), "Sandbox CRD not installed (agents.x-k8s.io/v1beta1)") {
			t.Fatalf("message = %q", err.Error())
		}
	})

	t.Run("no match", func(t *testing.T) {
		err := Translate(&meta.NoKindMatchError{GroupKind: schema.GroupKind{Group: SandboxGroup, Kind: SandboxKind}})
		if !errors.Is(err, ErrCRDMissing) {
			t.Fatalf("got %v, want ErrCRDMissing", err)
		}
	})

	t.Run("passthrough", func(t *testing.T) {
		orig := errors.New("connection refused")
		if Translate(orig) != orig {
			t.Fatal("expected original error")
		}
	})
}
