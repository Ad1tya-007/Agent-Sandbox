package watcher

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/resources"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/sandbox"
)

func TestSnapshotThenAddedUpdatedDeleted(t *testing.T) {
	m := newManager(nil, nil)
	events, unsub := m.Subscribe()
	defer unsub()

	m.handleAdd(testSandbox("alpha", "1"), true)
	m.handleAdd(testSandbox("beta", "1"), true)
	if got := recvNoEvent(t, events); got != nil {
		t.Fatalf("published before sync: %+v", *got)
	}

	m.markSynced()
	if ev := recvEvent(t, events); ev.Type != EventSynced {
		t.Fatalf("first event = %s, want synced", ev.Type)
	}
	if m.Connection().State != models.ConnectionConnected {
		t.Fatalf("state = %s", m.Connection().State)
	}
	if msg := m.Connection().Message; msg == nil || *msg != "Watching sandboxes" {
		t.Fatalf("message = %v", m.Connection().Message)
	}
	snap := m.Snapshot()
	if len(snap) != 2 || snap[0].Name != "alpha" || snap[1].Name != "beta" {
		t.Fatalf("snapshot = %+v", names(snap))
	}

	m.handleAdd(testSandbox("gamma", "1"), false)
	if ev := recvEvent(t, events); ev.Type != EventAdded || ev.Sandbox.Name != "gamma" {
		t.Fatalf("added = %+v", ev)
	}

	ready := testSandbox("gamma", "2")
	setReady(ready)
	m.handleUpdate(testSandbox("gamma", "1"), ready)
	if ev := recvEvent(t, events); ev.Type != EventUpdated || ev.Sandbox.Status != models.PhaseRunning {
		t.Fatalf("updated = %+v", ev)
	}

	m.handleDelete(ready)
	if ev := recvEvent(t, events); ev.Type != EventDeleted || ev.Name != "gamma" {
		t.Fatalf("deleted = %+v", ev)
	}

	late := m.Snapshot()
	if len(late) != 2 || late[0].Name != "alpha" || late[1].Name != "beta" {
		t.Fatalf("late snapshot = %+v", names(late))
	}
	if late == nil {
		t.Fatal("snapshot is nil")
	}
}

func TestSkipUnchangedResourceVersion(t *testing.T) {
	m := newManager(nil, nil)
	events, unsub := m.Subscribe()
	defer unsub()
	obj := testSandbox("demo", "7")
	m.handleAdd(obj, true)
	m.markSynced()
	_ = recvEvent(t, events)

	m.handleUpdate(obj, obj.DeepCopy())
	if got := recvNoEvent(t, events); got != nil {
		t.Fatalf("same resourceVersion published update: %+v", *got)
	}
}

func TestDeletedFinalStateUnknown(t *testing.T) {
	m := newManager(nil, nil)
	events, unsub := m.Subscribe()
	defer unsub()
	obj := testSandbox("demo", "1")
	m.handleAdd(obj, true)
	m.markSynced()
	_ = recvEvent(t, events)

	m.handleDelete(cache.DeletedFinalStateUnknown{Key: "default/demo", Obj: obj})
	if ev := recvEvent(t, events); ev.Type != EventDeleted || ev.Name != "demo" {
		t.Fatalf("deleted = %+v", ev)
	}
	if snap := m.Snapshot(); len(snap) != 0 {
		t.Fatalf("snapshot after delete = %+v", snap)
	}

	m.handleDelete(cache.DeletedFinalStateUnknown{Key: "default/ghost", Obj: nil})
	if ev := recvEvent(t, events); ev.Type != EventDeleted || ev.Name != "ghost" {
		t.Fatalf("deleted by key = %+v", ev)
	}
}

func TestEmptySnapshotIsNonNil(t *testing.T) {
	m := newManager(nil, nil)
	m.markSynced()
	snap := m.Snapshot()
	if snap == nil {
		t.Fatal("empty snapshot must be []")
	}
	if len(snap) != 0 {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestStartWithoutKubeconfig(t *testing.T) {
	k8s := kubernetes.New("/definitely/not-a-kubeconfig", "agents")
	m := New(k8s, nil)
	events, unsub := m.Subscribe()
	defer unsub()
	m.Start(context.Background())
	defer m.Stop()

	ev := recvEvent(t, events)
	if ev.Type != EventError {
		t.Fatalf("event = %s, want error", ev.Type)
	}
	conn := m.Connection()
	if conn.State != models.ConnectionError {
		t.Fatalf("state = %s", conn.State)
	}
	if conn.Cluster != nil {
		t.Fatalf("cluster = %v, want null", conn.Cluster)
	}
	if conn.Message == nil || *conn.Message == "" {
		t.Fatal("expected error message")
	}
}

func TestProjectUsesMapper(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "demo"},
		Spec:       corev1.PodSpec{NodeName: "worker-1"},
		Status:     corev1.PodStatus{PodIP: "10.0.0.8"},
	}
	m := newManager(nil, stubResolver{
		related: resources.Related{
			Pod: pod,
			Events: []models.TimelineEvent{{
				ID: "e1", Title: "Sandbox Created", Detail: "accepted", At: "2026-08-27T20:01:02Z",
			}},
		},
	})
	m.handleAdd(testSandbox("demo", "1"), true)
	snap := m.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot = %+v", snap)
	}
	if snap[0].Node == nil || *snap[0].Node != "worker-1" {
		t.Fatalf("node = %v", snap[0].Node)
	}
	if snap[0].IP == nil || *snap[0].IP != "10.0.0.8" {
		t.Fatalf("ip = %v", snap[0].IP)
	}
	if len(snap[0].Events) != 1 || snap[0].Events[0].Title != "Sandbox Created" {
		t.Fatalf("events = %+v", snap[0].Events)
	}
}

func TestInformerSnapshotThenDiffs(t *testing.T) {
	existing := testSandbox("alpha", "1")
	fw := watch.NewRaceFreeFake()
	lw := &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			list := &unstructured.UnstructuredList{}
			list.SetAPIVersion("agents.x-k8s.io/v1beta1")
			list.SetKind("SandboxList")
			list.SetResourceVersion("1")
			list.Items = []unstructured.Unstructured{*existing.DeepCopy()}
			return list, nil
		},
		WatchFunc: func(metav1.ListOptions) (watch.Interface, error) {
			return fw, nil
		},
	}

	m := newManager(nil, nil)
	m.clusterName = "kind-kind"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, unsub := m.Subscribe()
	defer unsub()
	go m.runInformer(ctx, lw)

	ev := recvEvent(t, events)
	if ev.Type != EventSynced {
		t.Fatalf("first event = %s, want synced", ev.Type)
	}
	snap := m.Snapshot()
	if len(snap) != 1 || snap[0].Name != "alpha" {
		t.Fatalf("snapshot = %+v", names(snap))
	}
	if m.Connection().State != models.ConnectionConnected {
		t.Fatalf("state = %s", m.Connection().State)
	}

	fw.Add(testSandbox("beta", "2"))
	if ev := recvEvent(t, events); ev.Type != EventAdded || ev.Sandbox.Name != "beta" {
		t.Fatalf("added = %+v", ev)
	}

	updated := testSandbox("beta", "3")
	setReady(updated)
	fw.Modify(updated)
	if ev := recvEvent(t, events); ev.Type != EventUpdated || ev.Sandbox.Name != "beta" || ev.Sandbox.Status != models.PhaseRunning {
		t.Fatalf("updated = %+v", ev)
	}

	same := updated.DeepCopy()
	fw.Modify(same)
	if got := recvNoEvent(t, events); got != nil {
		t.Fatalf("same RV published update: %+v", *got)
	}

	fw.Delete(updated)
	if ev := recvEvent(t, events); ev.Type != EventDeleted || ev.Name != "beta" {
		t.Fatalf("deleted = %+v", ev)
	}

	late := m.Snapshot()
	if len(late) != 1 || late[0].Name != "alpha" {
		t.Fatalf("late snapshot = %+v", names(late))
	}
}

func TestStartSyncsWhenClusterAvailable(t *testing.T) {
	k8s := kubernetes.New(os.Getenv("KUBECONFIG"), os.Getenv("AGENT_SANDBOX_NAMESPACE"))
	if k8s.Err() != nil {
		t.Skipf("no cluster: %v", k8s.Err())
	}
	m := New(k8s, resources.New(k8s))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	events, unsub := m.Subscribe()
	defer unsub()
	m.Start(ctx)
	defer m.Stop()

	ev := recvEventTimeout(t, events, 12*time.Second)
	switch ev.Type {
	case EventSynced:
		if m.Connection().State != models.ConnectionConnected {
			t.Fatalf("state = %s", m.Connection().State)
		}
		if m.Snapshot() == nil {
			t.Fatal("snapshot is nil")
		}
		t.Logf("synced %d sandbox(es) cluster=%v", len(m.Snapshot()), ptr(m.Connection().Cluster))
	case EventError:
		msg := ptr(m.Connection().Message)
		if !strings.Contains(msg, "Sandbox CRD not installed") {
			t.Fatalf("watch error: %s", msg)
		}
		t.Log(msg)
	default:
		t.Fatalf("unexpected event %s", ev.Type)
	}
}

func TestWatchErrorMessageForMissingCRD(t *testing.T) {
	msg := watchMessage(kubernetes.ErrCRDMissing)
	if msg != kubernetes.ErrCRDMissing.Error() {
		t.Fatalf("message = %q", msg)
	}
}

type stubResolver struct {
	related resources.Related
	err     error
}

func (s stubResolver) Resolve(context.Context, *unstructured.Unstructured) (resources.Related, error) {
	return s.related, s.err
}

func testSandbox(name, rv string) *unstructured.Unstructured {
	obj := sandbox.FromCreate("default", models.CreateInput{
		Name:   name,
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	})
	obj.SetResourceVersion(rv)
	obj.SetUID(types.UID("uid-" + name))
	return obj
}

func setReady(obj *unstructured.Unstructured) {
	_ = unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"type": "Ready", "status": "True", "message": "pod is ready"},
	}, "status", "conditions")
}

func recvEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	return recvEventTimeout(t, ch, 5*time.Second)
}

func recvEventTimeout(t *testing.T, ch <-chan Event, d time.Duration) Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(d):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func recvNoEvent(t *testing.T, ch <-chan Event) *Event {
	t.Helper()
	select {
	case ev := <-ch:
		return &ev
	case <-time.After(80 * time.Millisecond):
		return nil
	}
}

func names(sandboxes []models.Sandbox) []string {
	out := make([]string, len(sandboxes))
	for i, sb := range sandboxes {
		out[i] = sb.Name
	}
	return out
}

func ptr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestMarkErrorUsesFriendlyCRDMessage(t *testing.T) {
	m := newManager(nil, nil)
	events, unsub := m.Subscribe()
	defer unsub()
	m.markError(errors.Join(kubernetes.ErrCRDMissing, errors.New("no match")))
	ev := recvEvent(t, events)
	if ev.Type != EventError {
		t.Fatalf("type = %s", ev.Type)
	}
	if m.Connection().Message == nil || *m.Connection().Message != kubernetes.ErrCRDMissing.Error() {
		t.Fatalf("message = %v", m.Connection().Message)
	}
}
