package sandbox

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TestPhase(t *testing.T) {
	now := metav1.Now()
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
		want models.SandboxPhase
	}{
		{name: "nil", want: models.PhasePending},
		{name: "empty", obj: u(nil), want: models.PhasePending},
		{
			name: "terminating",
			obj: u(func(o *unstructured.Unstructured) {
				o.SetDeletionTimestamp(&now)
			}),
			want: models.PhaseTerminating,
		},
		{
			name: "paused operatingMode",
			obj: u(func(o *unstructured.Unstructured) {
				_ = unstructured.SetNestedField(o.Object, "Suspended", "spec", "operatingMode")
				setCondition(o, "Ready", "True", "")
			}),
			want: models.PhasePaused,
		},
		{
			name: "paused replicas 0",
			obj: u(func(o *unstructured.Unstructured) {
				_ = unstructured.SetNestedField(o.Object, int64(0), "spec", "replicas")
			}),
			want: models.PhasePaused,
		},
		{
			name: "failed finished PodFailed",
			obj: u(func(o *unstructured.Unstructured) {
				setCondition(o, "Finished", "True", "PodFailed")
			}),
			want: models.PhaseFailed,
		},
		{
			name: "failed ready false PodFailed",
			obj: u(func(o *unstructured.Unstructured) {
				setCondition(o, "Ready", "False", "PodFailed")
			}),
			want: models.PhaseFailed,
		},
		{
			name: "running",
			obj: u(func(o *unstructured.Unstructured) {
				setCondition(o, "Ready", "True", "")
			}),
			want: models.PhaseRunning,
		},
		{
			name: "pending not ready",
			obj: u(func(o *unstructured.Unstructured) {
				setCondition(o, "Ready", "False", "ContainersNotReady")
			}),
			want: models.PhasePending,
		},
		{
			name: "terminating beats paused",
			obj: u(func(o *unstructured.Unstructured) {
				o.SetDeletionTimestamp(&now)
				_ = unstructured.SetNestedField(o.Object, "Suspended", "spec", "operatingMode")
			}),
			want: models.PhaseTerminating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Phase(tt.obj); got != tt.want {
				t.Fatalf("Phase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func u(edit func(*unstructured.Unstructured)) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": sandboxAPIVersion,
		"kind":       sandboxKind,
		"metadata":   map[string]any{"name": "demo", "namespace": "default"},
		"spec":       map[string]any{"replicas": int64(1), "operatingMode": "Running"},
	}}
	if edit != nil {
		edit(obj)
	}
	return obj
}

func setCondition(obj *unstructured.Unstructured, typeName, status, reason string) {
	raw, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	raw = append(raw, map[string]any{
		"type":    typeName,
		"status":  status,
		"reason":  reason,
		"message": "",
	})
	_ = unstructured.SetNestedSlice(obj.Object, raw, "status", "conditions")
}
