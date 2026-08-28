package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore("agents")
	svc := New(store)
	in := models.CreateInput{
		Name:   "  research-agent  ",
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	}

	out, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if out == nil || out.Name != "research-agent" {
		t.Fatalf("result = %+v", out)
	}
	obj, ok := store.objects["research-agent"]
	if !ok {
		t.Fatal("sandbox not stored")
	}
	if obj.GetNamespace() != "agents" {
		t.Fatalf("namespace = %q", obj.GetNamespace())
	}
	service, found, _ := unstructured.NestedBool(obj.Object, "spec", "service")
	if !found || !service {
		t.Fatal("expected spec.service=true")
	}
}

func TestServiceCreateInvalid(t *testing.T) {
	store := newFakeStore("default")
	_, err := New(store).Create(context.Background(), models.CreateInput{
		Name: "My_Sandbox", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	requireError(t, err, models.ErrInvalidName)
	if len(store.objects) != 0 {
		t.Fatal("invalid create must not hit the store")
	}
}

func TestServiceCreateAlreadyExists(t *testing.T) {
	store := newFakeStore("default")
	store.put(runningSandbox("research-agent"))
	_, err := New(store).Create(context.Background(), models.CreateInput{
		Name: "research-agent", Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi",
	})
	requireError(t, err, models.ErrAlreadyExists.Format("research-agent"))
}

func TestServicePauseRunning(t *testing.T) {
	store := newFakeStore("default")
	store.put(runningSandbox("demo"))
	if err := New(store).Pause(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if len(store.patches) != 1 {
		t.Fatalf("patches = %d, want 1", len(store.patches))
	}
	spec := patchSpec(t, store.patches[0])
	if spec["operatingMode"] != "Suspended" {
		t.Fatalf("patch spec = %v", spec)
	}
	if _, ok := spec["replicas"]; ok {
		t.Fatal("first successful patch should not include replicas")
	}
}

func TestServicePauseFallsBackToReplicas(t *testing.T) {
	store := newFakeStore("default")
	store.put(runningSandbox("demo"))
	store.failPatch = 1
	if err := New(store).Pause(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if len(store.patches) != 2 {
		t.Fatalf("patches = %d, want 2", len(store.patches))
	}
	spec := patchSpec(t, store.patches[1])
	if replicas, _ := spec["replicas"].(float64); replicas != 0 {
		t.Fatalf("fallback patch = %v", spec)
	}
}

func TestServicePauseNotRunning(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
	}{
		{name: "pending", obj: FromCreate("default", validInput("demo"))},
		{name: "paused", obj: pausedSandbox("demo")},
		{name: "failed", obj: failedSandbox("demo")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore("default")
			store.put(tt.obj)
			err := New(store).Pause(context.Background(), "demo")
			requireError(t, err, models.ErrPauseNotRunning)
			if len(store.patches) != 0 {
				t.Fatalf("patches = %d", len(store.patches))
			}
		})
	}
}

func TestServicePauseNotFound(t *testing.T) {
	err := New(newFakeStore("default")).Pause(context.Background(), "missing")
	requireError(t, err, models.ErrMissing.Format("missing"))
}

func TestServiceResumePaused(t *testing.T) {
	store := newFakeStore("default")
	store.put(pausedSandbox("demo"))
	if err := New(store).Resume(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	spec := patchSpec(t, store.patches[0])
	if spec["operatingMode"] != "Running" {
		t.Fatalf("patch spec = %v", spec)
	}
}

func TestServiceResumeFallsBackToReplicas(t *testing.T) {
	store := newFakeStore("default")
	store.put(pausedSandbox("demo"))
	store.failPatch = 1
	if err := New(store).Resume(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	spec := patchSpec(t, store.patches[1])
	if replicas, _ := spec["replicas"].(float64); replicas != 1 {
		t.Fatalf("fallback patch = %v", spec)
	}
}

func TestServiceResumeNotPaused(t *testing.T) {
	tests := []struct {
		name string
		obj  *unstructured.Unstructured
	}{
		{name: "pending", obj: FromCreate("default", validInput("demo"))},
		{name: "running", obj: runningSandbox("demo")},
		{name: "failed", obj: failedSandbox("demo")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore("default")
			store.put(tt.obj)
			err := New(store).Resume(context.Background(), "demo")
			requireError(t, err, models.ErrResumeNotPaused)
			if len(store.patches) != 0 {
				t.Fatalf("patches = %d", len(store.patches))
			}
		})
	}
}

func TestServiceDelete(t *testing.T) {
	store := newFakeStore("default")
	store.put(runningSandbox("demo"))
	if err := New(store).Delete(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.objects["demo"]; ok {
		t.Fatal("object still stored")
	}
}

func TestServiceDeleteNotFound(t *testing.T) {
	store := newFakeStore("default")
	err := New(store).Delete(context.Background(), "missing")
	requireError(t, err, models.ErrMissing.Format("missing"))
	if len(store.deletes) != 0 {
		t.Fatal("delete must not run when get 404s")
	}
}

func TestServiceMapsInternalErrors(t *testing.T) {
	store := newFakeStore("default")
	store.failCreate = errors.New("apiserver down")
	_, err := New(store).Create(context.Background(), validInput("demo"))
	requireError(t, err, models.Internal("apiserver down"))
}

func validInput(name string) models.CreateInput {
	return models.CreateInput{Name: name, Image: "python:3.12-slim", CPU: "500m", Memory: "1Gi"}
}

func runningSandbox(name string) *unstructured.Unstructured {
	obj := FromCreate("default", validInput(name))
	setCondition(obj, "Ready", "True", "")
	return obj
}

func pausedSandbox(name string) *unstructured.Unstructured {
	obj := FromCreate("default", validInput(name))
	_ = unstructured.SetNestedField(obj.Object, "Suspended", "spec", "operatingMode")
	return obj
}

func failedSandbox(name string) *unstructured.Unstructured {
	obj := FromCreate("default", validInput(name))
	setCondition(obj, "Ready", "False", "PodFailed")
	return obj
}

type fakeStore struct {
	ns         string
	objects    map[string]*unstructured.Unstructured
	patches    [][]byte
	deletes    []string
	failPatch  int
	failCreate error
	failDelete error
}

func newFakeStore(ns string) *fakeStore {
	return &fakeStore{ns: ns, objects: map[string]*unstructured.Unstructured{}}
}

var _ Store = (*fakeStore)(nil)

func (f *fakeStore) put(obj *unstructured.Unstructured) {
	f.objects[obj.GetName()] = obj.DeepCopy()
}

func (f *fakeStore) Namespace() string { return f.ns }

func (f *fakeStore) CreateSandbox(_ context.Context, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	if f.failCreate != nil {
		return nil, f.failCreate
	}
	name := obj.GetName()
	if _, ok := f.objects[name]; ok {
		return nil, kubernetes.ErrAlreadyExists
	}
	clone := obj.DeepCopy()
	f.objects[name] = clone
	return clone.DeepCopy(), nil
}

func (f *fakeStore) GetSandbox(_ context.Context, name string) (*unstructured.Unstructured, error) {
	obj, ok := f.objects[name]
	if !ok {
		return nil, kubernetes.ErrNotFound
	}
	return obj.DeepCopy(), nil
}

func (f *fakeStore) PatchSandbox(_ context.Context, name string, patch []byte) (*unstructured.Unstructured, error) {
	f.patches = append(f.patches, append([]byte(nil), patch...))
	obj, ok := f.objects[name]
	if !ok {
		return nil, kubernetes.ErrNotFound
	}
	if f.failPatch > 0 {
		f.failPatch--
		return nil, errors.New("patch not accepted")
	}
	return obj.DeepCopy(), nil
}

func (f *fakeStore) DeleteSandbox(_ context.Context, name string) error {
	f.deletes = append(f.deletes, name)
	if f.failDelete != nil {
		return f.failDelete
	}
	if _, ok := f.objects[name]; !ok {
		return kubernetes.ErrNotFound
	}
	delete(f.objects, name)
	return nil
}

func patchSpec(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	spec, _ := m["spec"].(map[string]any)
	if spec == nil {
		t.Fatalf("no spec in %s", raw)
	}
	return spec
}

func requireError(t *testing.T, err error, want *models.Error) {
	t.Helper()
	var got *models.Error
	if !models.AsError(err, &got) {
		t.Fatalf("err = %v (%T), want %+v", err, err, want)
	}
	if got.Kind != want.Kind || got.Message != want.Message {
		t.Fatalf("err = %+v, want %+v", got, want)
	}
}
