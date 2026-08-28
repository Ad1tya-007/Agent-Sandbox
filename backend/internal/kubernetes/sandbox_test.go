package kubernetes

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func testClient(t *testing.T) *Client {
	t.Helper()
	gvr := SandboxGVR()
	scheme := runtime.NewScheme()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{
		gvr: "SandboxList",
	})
	return &Client{
		kube:      fake.NewSimpleClientset(),
		dynamic:   dyn,
		namespace: "default",
		gvr:       gvr,
	}
}

func sandboxObj(name string, mode string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": SandboxGroup + "/" + SandboxVersion,
		"kind":       SandboxKind,
		"metadata": map[string]any{
			"name":      name,
			"namespace": "default",
		},
		"spec": map[string]any{
			"operatingMode": mode,
		},
	}}
}

func TestSandboxGVR(t *testing.T) {
	gvr := SandboxGVR()
	if gvr.Group != "agents.x-k8s.io" || gvr.Version != "v1beta1" || gvr.Resource != "sandboxes" {
		t.Fatalf("SandboxGVR() = %+v", gvr)
	}
	c := testClient(t)
	if c.SandboxGVR() != gvr {
		t.Fatalf("client GVR = %+v", c.SandboxGVR())
	}
}

func TestSandboxCRUD(t *testing.T) {
	ctx := context.Background()
	c := testClient(t)

	created, err := c.CreateSandbox(ctx, sandboxObj("demo", "Running"))
	if err != nil {
		t.Fatal(err)
	}
	if created.GetName() != "demo" {
		t.Fatalf("created name = %q", created.GetName())
	}

	got, err := c.GetSandbox(ctx, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if got.GetName() != "demo" {
		t.Fatalf("get name = %q", got.GetName())
	}

	patched, err := c.PatchSandbox(ctx, "demo", []byte(`{"spec":{"operatingMode":"Suspended"}}`))
	if err != nil {
		t.Fatal(err)
	}
	mode, found, err := unstructured.NestedString(patched.Object, "spec", "operatingMode")
	if err != nil || !found || mode != "Suspended" {
		t.Fatalf("operatingMode after patch = %q found=%v err=%v", mode, found, err)
	}

	if err := c.DeleteSandbox(ctx, "demo"); err != nil {
		t.Fatal(err)
	}
	_, err = c.GetSandbox(ctx, "demo")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete = %v, want ErrNotFound", err)
	}
}

func TestCreateSandboxAlreadyExists(t *testing.T) {
	ctx := context.Background()
	c := testClient(t)
	if _, err := c.CreateSandbox(ctx, sandboxObj("demo", "Running")); err != nil {
		t.Fatal(err)
	}
	_, err := c.CreateSandbox(ctx, sandboxObj("demo", "Running"))
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("second create = %v, want ErrAlreadyExists", err)
	}
}

func TestGetSandboxNotFound(t *testing.T) {
	_, err := testClient(t).GetSandbox(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetSandbox = %v, want ErrNotFound", err)
	}
}

func TestSandboxMethodsUnready(t *testing.T) {
	ctx := context.Background()
	c := New("/definitely/not/a/kubeconfig", "ns")
	if _, err := c.CreateSandbox(ctx, sandboxObj("demo", "Running")); err == nil {
		t.Fatal("CreateSandbox: expected error")
	}
	if _, err := c.GetSandbox(ctx, "demo"); err == nil {
		t.Fatal("GetSandbox: expected error")
	}
	if err := c.DeleteSandbox(ctx, "demo"); err == nil {
		t.Fatal("DeleteSandbox: expected error")
	}
	if _, err := c.PatchSandbox(ctx, "demo", []byte(`{}`)); err == nil {
		t.Fatal("PatchSandbox: expected error")
	}
}

func TestOpenLogsUnready(t *testing.T) {
	c := New("/definitely/not/a/kubeconfig", "")
	_, err := c.OpenLogs(context.Background(), "default", "pod", "sandbox", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCoreLists(t *testing.T) {
	ctx := context.Background()
	c := testClient(t)

	_, err := c.kube.CoreV1().Pods("default").Create(ctx, corev1Pod("demo", map[string]string{"app": "demo"}), metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.kube.CoreV1().PersistentVolumeClaims("default").Create(ctx, corev1PVC("workspace", map[string]string{"app": "demo"}), metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.kube.CoreV1().Services("default").Create(ctx, corev1Service("demo", map[string]string{"app": "demo"}), metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.kube.CoreV1().Events("default").Create(ctx, corev1Event("demo-evt", "demo"), metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}

	pods, err := c.ListPods(ctx, "default", "app=demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(pods.Items) != 1 || pods.Items[0].Name != "demo" {
		t.Fatalf("pods = %+v", pods.Items)
	}

	pod, err := c.GetPod(ctx, "default", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if pod.Name != "demo" {
		t.Fatalf("pod = %q", pod.Name)
	}

	pvcs, err := c.ListPVCs(ctx, "default", "app=demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(pvcs.Items) != 1 {
		t.Fatalf("pvcs = %d", len(pvcs.Items))
	}

	svcs, err := c.ListServices(ctx, "default", "app=demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs.Items) != 1 {
		t.Fatalf("services = %d", len(svcs.Items))
	}

	events, err := c.ListEvents(ctx, "default", "involvedObject.name=demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Items) != 1 {
		t.Fatalf("events = %d", len(events.Items))
	}

	_, err = c.GetPod(ctx, "default", "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetPod missing = %v, want ErrNotFound", err)
	}

	pvc, err := c.GetPVC(ctx, "default", "workspace")
	if err != nil {
		t.Fatal(err)
	}
	if pvc.Name != "workspace" {
		t.Fatalf("pvc = %q", pvc.Name)
	}
	svc, err := c.GetService(ctx, "default", "demo")
	if err != nil {
		t.Fatal(err)
	}
	if svc.Name != "demo" {
		t.Fatalf("service = %q", svc.Name)
	}
}

func TestCoreMethodsUnready(t *testing.T) {
	ctx := context.Background()
	c := New("/definitely/not/a/kubeconfig", "")
	if _, err := c.ListPods(ctx, "default", "app=demo"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.GetPod(ctx, "default", "demo"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.GetPVC(ctx, "default", "workspace"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.GetService(ctx, "default", "demo"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.ListPVCs(ctx, "default", ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.ListServices(ctx, "default", ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.ListEvents(ctx, "default", ""); err == nil {
		t.Fatal("expected error")
	}
}

func corev1Pod(name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: labels}}
}

func corev1PVC(name string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: labels}}
}

func corev1Service(name string, labels map[string]string) *corev1.Service {
	return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: labels}}
}

func corev1Event(name, involved string) *corev1.Event {
	return &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Name: name, Namespace: "default"},
		InvolvedObject: corev1.ObjectReference{Name: involved, Kind: SandboxKind},
	}
}
