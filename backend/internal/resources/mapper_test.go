package resources

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
)

func TestResolveOwnedResources(t *testing.T) {
	uid := types.UID("sb-1")
	core := &fakeCore{
		t: t,
		pods: []corev1.Pod{
			pod("other", "default", map[string]string{"app": "demo"}, nil, ""),
			pod("demo", "default", map[string]string{"app": "demo"}, []metav1.OwnerReference{sandboxOwner("demo", uid)}, "10.0.0.8"),
		},
		pvcs: []corev1.PersistentVolumeClaim{
			pvc("workspace", "default", map[string]string{"app": "demo"}, []metav1.OwnerReference{sandboxOwner("demo", uid)}),
			pvc("unrelated", "default", map[string]string{"app": "demo"}, nil),
		},
		svcs: []corev1.Service{
			svc("demo", "default", map[string]string{"app": "demo"}, []metav1.OwnerReference{sandboxOwner("demo", uid)}),
		},
		events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{UID: "e1", Name: "demo.1", Namespace: "default"},
				Reason:         "Created",
				InvolvedObject: corev1.ObjectReference{Kind: "Sandbox", Name: "demo"},
			},
			{
				ObjectMeta:     metav1.ObjectMeta{UID: "e2", Name: "demo.2", Namespace: "default"},
				Reason:         "Started",
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "demo"},
			},
		},
	}
	obj := sandboxCR("demo", "default", uid, "app=demo", "demo", []string{"workspace"})
	got, err := New(core).Resolve(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if got.Pod == nil || got.Pod.Name != "demo" || got.Pod.Status.PodIP != "10.0.0.8" {
		t.Fatalf("pod = %+v", got.Pod)
	}
	if len(got.PVCs) != 1 || got.PVCs[0].Name != "workspace" {
		t.Fatalf("pvcs = %+v", got.PVCs)
	}
	if got.Service == nil || got.Service.Name != "demo" {
		t.Fatalf("service = %+v", got.Service)
	}
	if len(got.Events) != 2 {
		t.Fatalf("events = %+v", got.Events)
	}
	if got.Events[0].Title != "Sandbox Created" || got.Events[1].Title != "Container Started" {
		t.Fatalf("event titles = %+v", got.Events)
	}
}

func TestResolveServiceByStatusName(t *testing.T) {
	core := &fakeCore{
		t: t,
		svcs: []corev1.Service{
			svc("demo-svc", "default", nil, nil),
		},
	}
	obj := sandboxCR("demo", "default", "sb-1", "", "demo-svc", nil)
	got, err := New(core).Resolve(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if got.Service == nil || got.Service.Name != "demo-svc" {
		t.Fatalf("service = %+v", got.Service)
	}
	for _, call := range core.listCalls {
		if strings.HasPrefix(call, "pods:") || strings.HasPrefix(call, "pvcs:") || strings.HasPrefix(call, "services:") {
			t.Fatalf("unexpected resource list: %v", core.listCalls)
		}
	}
}

func TestResolvePodByNameFallback(t *testing.T) {
	core := &fakeCore{
		t: t,
		pods: []corev1.Pod{
			pod("demo", "default", nil, nil, "10.1.2.3"),
		},
	}
	obj := sandboxCR("demo", "default", "sb-1", "", "", nil)
	got, err := New(core).Resolve(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if got.Pod == nil || got.Pod.Name != "demo" {
		t.Fatalf("pod = %+v", got.Pod)
	}
}

func TestResolvePVCFromVolumeClaimTemplate(t *testing.T) {
	core := &fakeCore{
		t: t,
		pvcs: []corev1.PersistentVolumeClaim{
			pvc("workspace", "default", nil, nil),
		},
	}
	obj := sandboxCR("demo", "default", "sb-1", "", "", []string{"workspace"})
	got, err := New(core).Resolve(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PVCs) != 1 || got.PVCs[0].Name != "workspace" {
		t.Fatalf("pvcs = %+v", got.PVCs)
	}
}

func TestResolveEventsIncludePodName(t *testing.T) {
	uid := types.UID("sb-1")
	core := &fakeCore{
		t: t,
		pods: []corev1.Pod{
			pod("demo-pod", "default", map[string]string{"app": "demo"}, []metav1.OwnerReference{sandboxOwner("demo", uid)}, ""),
		},
		events: []corev1.Event{
			{
				ObjectMeta:     metav1.ObjectMeta{UID: "e1", Name: "sandbox-evt"},
				Reason:         "Created",
				InvolvedObject: corev1.ObjectReference{Kind: "Sandbox", Name: "demo"},
			},
			{
				ObjectMeta:     metav1.ObjectMeta{UID: "e2", Name: "pod-evt"},
				Reason:         "Pulled",
				Message:        `Successfully pulled image "python:3.12-slim"`,
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "demo-pod"},
			},
		},
	}
	obj := sandboxCR("demo", "default", uid, "app=demo", "", nil)
	got, err := New(core).Resolve(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 2 {
		t.Fatalf("events = %+v", got.Events)
	}
}

func TestResolveNilInputs(t *testing.T) {
	got, err := New(nil).Resolve(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Events == nil || got.PVCs == nil {
		t.Fatalf("empty slices must be non-nil: %+v", got)
	}
}

func TestStatusSelectorFromMap(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"selector": map[string]any{"agents.x-k8s.io/sandbox": "demo"},
		},
	}}
	got := statusSelector(obj)
	sel, err := labels.Parse(got)
	if err != nil || !sel.Matches(labels.Set{"agents.x-k8s.io/sandbox": "demo"}) {
		t.Fatalf("selector = %q", got)
	}
}

func sandboxCR(name, ns string, uid types.UID, selector, service string, claims []string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": kubernetes.SandboxGroup + "/" + kubernetes.SandboxVersion,
		"kind":       kubernetes.SandboxKind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": ns,
			"uid":       string(uid),
		},
		"spec":   map[string]any{},
		"status": map[string]any{},
	}}
	if selector != "" {
		_ = unstructured.SetNestedField(obj.Object, selector, "status", "selector")
	}
	if service != "" {
		_ = unstructured.SetNestedField(obj.Object, service, "status", "service")
	}
	if len(claims) > 0 {
		var raw []any
		for _, c := range claims {
			raw = append(raw, map[string]any{"metadata": map[string]any{"name": c}})
		}
		_ = unstructured.SetNestedSlice(obj.Object, raw, "spec", "volumeClaimTemplates")
	}
	return obj
}

func sandboxOwner(name string, uid types.UID) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: kubernetes.SandboxGroup + "/" + kubernetes.SandboxVersion,
		Kind:       kubernetes.SandboxKind,
		Name:       name,
		UID:        uid,
	}
}

func pod(name, ns string, ls map[string]string, owners []metav1.OwnerReference, ip string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: ls, OwnerReferences: owners},
		Status:     corev1.PodStatus{PodIP: ip},
	}
}

func pvc(name, ns string, ls map[string]string, owners []metav1.OwnerReference) corev1.PersistentVolumeClaim {
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: ls, OwnerReferences: owners},
	}
}

func svc(name, ns string, ls map[string]string, owners []metav1.OwnerReference) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, Labels: ls, OwnerReferences: owners},
	}
}

type fakeCore struct {
	t         *testing.T
	pods      []corev1.Pod
	pvcs      []corev1.PersistentVolumeClaim
	svcs      []corev1.Service
	events    []corev1.Event
	listCalls []string
}

func (f *fakeCore) ListPods(_ context.Context, namespace, selector string) (*corev1.PodList, error) {
	f.recordList("pods", selector)
	out := &corev1.PodList{}
	sel := mustSelector(f.t, selector)
	for _, p := range f.pods {
		if p.Namespace == namespace && sel.Matches(labels.Set(p.Labels)) {
			out.Items = append(out.Items, p)
		}
	}
	return out, nil
}

func (f *fakeCore) GetPod(_ context.Context, namespace, name string) (*corev1.Pod, error) {
	for i := range f.pods {
		if f.pods[i].Namespace == namespace && f.pods[i].Name == name {
			return f.pods[i].DeepCopy(), nil
		}
	}
	return nil, kubernetes.ErrNotFound
}

func (f *fakeCore) ListPVCs(_ context.Context, namespace, selector string) (*corev1.PersistentVolumeClaimList, error) {
	f.recordList("pvcs", selector)
	out := &corev1.PersistentVolumeClaimList{}
	sel := mustSelector(f.t, selector)
	for _, p := range f.pvcs {
		if p.Namespace == namespace && sel.Matches(labels.Set(p.Labels)) {
			out.Items = append(out.Items, p)
		}
	}
	return out, nil
}

func (f *fakeCore) GetPVC(_ context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	for i := range f.pvcs {
		if f.pvcs[i].Namespace == namespace && f.pvcs[i].Name == name {
			return f.pvcs[i].DeepCopy(), nil
		}
	}
	return nil, kubernetes.ErrNotFound
}

func (f *fakeCore) ListServices(_ context.Context, namespace, selector string) (*corev1.ServiceList, error) {
	f.recordList("services", selector)
	out := &corev1.ServiceList{}
	sel := mustSelector(f.t, selector)
	for _, s := range f.svcs {
		if s.Namespace == namespace && sel.Matches(labels.Set(s.Labels)) {
			out.Items = append(out.Items, s)
		}
	}
	return out, nil
}

func (f *fakeCore) GetService(_ context.Context, namespace, name string) (*corev1.Service, error) {
	for i := range f.svcs {
		if f.svcs[i].Namespace == namespace && f.svcs[i].Name == name {
			return f.svcs[i].DeepCopy(), nil
		}
	}
	return nil, kubernetes.ErrNotFound
}

func (f *fakeCore) ListEvents(_ context.Context, namespace, fieldSelector string) (*corev1.EventList, error) {
	if fieldSelector == "" {
		f.t.Error("ListEvents with empty field selector")
	}
	f.listCalls = append(f.listCalls, "events:"+fieldSelector)
	want := ""
	if _, rest, ok := splitEq(fieldSelector); ok {
		want = rest
	}
	out := &corev1.EventList{}
	for _, ev := range f.events {
		if ev.Namespace != "" && ev.Namespace != namespace {
			continue
		}
		if ev.InvolvedObject.Name == want {
			out.Items = append(out.Items, ev)
		}
	}
	return out, nil
}

func (f *fakeCore) recordList(kind, selector string) {
	if selector == "" {
		f.t.Errorf("List%s with empty selector", kind)
	}
	f.listCalls = append(f.listCalls, kind+":"+selector)
}

func mustSelector(t *testing.T, raw string) labels.Selector {
	t.Helper()
	sel, err := labels.Parse(raw)
	if err != nil {
		t.Fatalf("selector %q: %v", raw, err)
	}
	return sel
}

func splitEq(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
