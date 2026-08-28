package sandbox

import (
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TestProjectFromCreate(t *testing.T) {
	in := models.CreateInput{
		Name:   "research-agent",
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	}
	obj := FromCreate("default", in)
	obj.SetCreationTimestamp(metav1.NewTime(time.Date(2026, 8, 27, 20, 1, 2, 0, time.UTC)))

	got := Project(obj, Related{})
	if got.Name != "research-agent" || got.Namespace != "default" {
		t.Fatalf("identity = %s/%s", got.Namespace, got.Name)
	}
	if got.Status != models.PhasePending {
		t.Fatalf("status = %q", got.Status)
	}
	if got.Image != "python:3.12-slim" || got.CPU != "500m" || got.Memory != "1Gi" {
		t.Fatalf("resources = %s %s %s", got.Image, got.CPU, got.Memory)
	}
	if got.PersistentStorage {
		t.Fatal("persistentStorage = true")
	}
	if got.Node != nil || got.IP != nil {
		t.Fatalf("node/ip = %v %v", got.Node, got.IP)
	}
	if got.CreatedAt != "2026-08-27T20:01:02Z" {
		t.Fatalf("createdAt = %q", got.CreatedAt)
	}
	if got.Conditions == nil || len(got.Conditions) != 0 {
		t.Fatalf("conditions = %#v", got.Conditions)
	}
	if got.Events == nil || len(got.Events) != 0 {
		t.Fatalf("events = %#v", got.Events)
	}
	if !strings.Contains(got.YAML, "kind: Sandbox") {
		t.Fatalf("yaml missing kind: %s", got.YAML)
	}
	if strings.Contains(got.YAML, "managedFields") {
		t.Fatalf("yaml still has managedFields: %s", got.YAML)
	}
}

func TestProjectPersistentStorageAndConditions(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name:              "research-agent",
		Image:             "python:3.12-slim",
		CPU:               "500m",
		Memory:            "1Gi",
		PersistentStorage: true,
	})
	_ = unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"type": "Ready", "status": "True", "message": "pod is ready"},
		map[string]any{"type": "Available", "status": "False", "message": "not yet"},
		map[string]any{"type": "Odd", "status": "maybe"},
	}, "status", "conditions")

	got := Project(obj, Related{})
	if !got.PersistentStorage {
		t.Fatal("persistentStorage = false")
	}
	if got.Status != models.PhaseRunning {
		t.Fatalf("status = %q", got.Status)
	}
	if len(got.Conditions) != 3 {
		t.Fatalf("conditions = %#v", got.Conditions)
	}
	if got.Conditions[0] != (models.Condition{Type: "Ready", Status: models.ConditionTrue, Message: "pod is ready"}) {
		t.Fatalf("ready = %+v", got.Conditions[0])
	}
	if got.Conditions[1].Status != models.ConditionFalse {
		t.Fatalf("available = %+v", got.Conditions[1])
	}
	if got.Conditions[2].Status != models.ConditionUnknown {
		t.Fatalf("odd = %+v", got.Conditions[2])
	}
}

func TestProjectNodeIPFromStatus(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	_ = unstructured.SetNestedField(obj.Object, "worker-1", "status", "nodeName")
	_ = unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"ip": "10.0.0.8"},
	}, "status", "podIPs")

	pod := &corev1.Pod{
		Spec:   corev1.PodSpec{NodeName: "other-node"},
		Status: corev1.PodStatus{PodIP: "10.9.9.9"},
	}
	got := Project(obj, Related{Pod: pod})
	if got.Node == nil || *got.Node != "worker-1" {
		t.Fatalf("node = %v", got.Node)
	}
	if got.IP == nil || *got.IP != "10.0.0.8" {
		t.Fatalf("ip = %v", got.IP)
	}
}

func TestProjectNodeIPFromPodFallback(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	pod := &corev1.Pod{
		Spec:   corev1.PodSpec{NodeName: "worker-2"},
		Status: corev1.PodStatus{PodIP: "10.0.0.9"},
	}
	got := Project(obj, Related{Pod: pod})
	if got.Node == nil || *got.Node != "worker-2" {
		t.Fatalf("node = %v", got.Node)
	}
	if got.IP == nil || *got.IP != "10.0.0.9" {
		t.Fatalf("ip = %v", got.IP)
	}
}

func TestProjectYAMLStripsManagedFields(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	_ = unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"manager": "kubectl", "operation": "Update"},
	}, "metadata", "managedFields")

	got := Project(obj, Related{})
	if strings.Contains(got.YAML, "managedFields") || strings.Contains(got.YAML, "kubectl") {
		t.Fatalf("yaml still has managedFields: %s", got.YAML)
	}
	_, found, _ := unstructured.NestedSlice(obj.Object, "metadata", "managedFields")
	if !found {
		t.Fatal("Project mutated the original object")
	}
}

func TestProjectEventsNeverNull(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	events := []models.TimelineEvent{{
		ID: "e1", Title: "Sandbox Created", Detail: "accepted", At: "2026-08-27T20:01:02Z",
	}}
	got := Project(obj, Related{Events: events})
	if len(got.Events) != 1 || got.Events[0].Title != "Sandbox Created" {
		t.Fatalf("events = %#v", got.Events)
	}

	nilObj := Project(nil, Related{})
	if nilObj.Events == nil || len(nilObj.Events) != 0 {
		t.Fatalf("nil obj events = %#v", nilObj.Events)
	}
	if nilObj.Conditions == nil {
		t.Fatal("nil obj conditions is nil")
	}
	if nilObj.Status != models.PhasePending {
		t.Fatalf("nil obj status = %q", nilObj.Status)
	}
}

func TestProjectStringPodIPs(t *testing.T) {
	obj := FromCreate("default", models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	_ = unstructured.SetNestedSlice(obj.Object, []any{"10.1.2.3"}, "status", "podIPs")
	got := Project(obj, Related{})
	if got.IP == nil || *got.IP != "10.1.2.3" {
		t.Fatalf("ip = %v", got.IP)
	}
}
