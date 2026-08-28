package logs

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/resources"
)

func TestSubscribeReceivesParsedLines(t *testing.T) {
	pr, pw := io.Pipe()
	cluster := newFakeCluster(pr)
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()

	snap, lines, stop := mgr.Subscribe("demo")
	defer stop()
	if snap == nil || len(snap) != 0 {
		t.Fatalf("snapshot = %#v", snap)
	}

	waitOpen(t, cluster)
	if _, err := io.WriteString(pw, "2026-08-27T20:01:02.123456789Z hello\n"); err != nil {
		t.Fatal(err)
	}

	got := recvLine(t, lines)
	if got.Message != "hello" || got.TS != "2026-08-27T20:01:02.123456789Z" || got.ID == "" {
		t.Fatalf("line = %+v", got)
	}
	cluster.mu.Lock()
	n := cluster.openN
	cluster.mu.Unlock()
	if n != 1 {
		t.Fatalf("opens = %d", n)
	}
	call := cluster.lastOpenMeta()
	if call.pod != "demo" || call.container != "sandbox" || call.since != nil {
		t.Fatalf("open = %+v", call)
	}
}

func TestTwoSubscribersReceiveTheSameLine(t *testing.T) {
	pr, pw := io.Pipe()
	cluster := newFakeCluster(pr)
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()

	_, a, stopA := mgr.Subscribe("demo")
	defer stopA()
	_, b, stopB := mgr.Subscribe("demo")
	defer stopB()

	waitOpen(t, cluster)
	if cluster.openCount() != 1 {
		t.Fatalf("opens = %d, want 1 upstream", cluster.openCount())
	}
	if _, err := io.WriteString(pw, "2026-08-27T20:01:02Z shared\n"); err != nil {
		t.Fatal(err)
	}
	if recvLine(t, a).Message != "shared" {
		t.Fatal("subscriber a missed line")
	}
	if recvLine(t, b).Message != "shared" {
		t.Fatal("subscriber b missed line")
	}
}

func TestLateSubscriberGetsBufferedSnapshot(t *testing.T) {
	pr, pw := io.Pipe()
	cluster := newFakeCluster(pr)
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()

	_, live, stopLive := mgr.Subscribe("demo")
	defer stopLive()
	waitOpen(t, cluster)
	if _, err := io.WriteString(pw, "2026-08-27T20:01:02Z first\n"); err != nil {
		t.Fatal(err)
	}
	if recvLine(t, live).Message != "first" {
		t.Fatal("expected live line")
	}

	snap, _, stopLate := mgr.Subscribe("demo")
	defer stopLate()
	if len(snap) != 1 || snap[0].Message != "first" {
		t.Fatalf("snapshot = %#v", snap)
	}
}

func TestUnsubscribeClosesUpstream(t *testing.T) {
	pr, pw := io.Pipe()
	closed := make(chan struct{})
	rc := &notifyCloser{ReadCloser: pr, closed: closed}
	cluster := newFakeCluster(rc)
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()

	_, _, stopA := mgr.Subscribe("demo")
	_, _, stopB := mgr.Subscribe("demo")
	waitOpen(t, cluster)

	stopA()
	select {
	case <-closed:
		t.Fatal("upstream closed while a subscriber remains")
	case <-time.After(50 * time.Millisecond):
	}

	stopB()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream not closed after last subscriber left")
	}
	_ = pw.Close()
}

func TestReconnectUsesSinceTime(t *testing.T) {
	firstR, firstW := io.Pipe()
	secondR, secondW := io.Pipe()
	cluster := &fakeCluster{
		ns:      "default",
		sandbox: sandboxObj("demo"),
		readers: []io.ReadCloser{firstR, secondR},
		opens:   make(chan openCall, 4),
	}
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()

	_, lines, stop := mgr.Subscribe("demo")
	defer stop()
	waitOpen(t, cluster)

	if _, err := io.WriteString(firstW, "2026-08-27T20:01:02Z one\n"); err != nil {
		t.Fatal(err)
	}
	if recvLine(t, lines).Message != "one" {
		t.Fatal("missing first line")
	}
	_ = firstW.Close()

	call := waitOpen(t, cluster)
	if call.since == nil {
		t.Fatal("reconnect should pass SinceTime")
	}
	if !call.since.Time.Equal(time.Date(2026, 8, 27, 20, 1, 2, 0, time.UTC)) {
		t.Fatalf("since = %v", call.since)
	}
	if _, err := io.WriteString(secondW, "2026-08-27T20:01:03Z two\n"); err != nil {
		t.Fatal(err)
	}
	if recvLine(t, lines).Message != "two" {
		t.Fatal("missing reconnected line")
	}
}

func TestEmptySnapshotUntilPodExists(t *testing.T) {
	pr, pw := io.Pipe()
	cluster := newFakeCluster(pr)
	resolver := &fakeResolver{}
	mgr := newManager(cluster, resolver)
	defer mgr.Close()

	snap, lines, stop := mgr.Subscribe("demo")
	defer stop()
	if snap == nil || len(snap) != 0 {
		t.Fatalf("snapshot = %#v", snap)
	}
	select {
	case <-cluster.opens:
		t.Fatal("opened logs before a pod existed")
	case <-time.After(80 * time.Millisecond):
	}

	resolver.set(testPod("demo"))
	waitOpen(t, cluster)
	if _, err := io.WriteString(pw, "2026-08-27T20:01:02Z ready\n"); err != nil {
		t.Fatal(err)
	}
	if recvLine(t, lines).Message != "ready" {
		t.Fatal("expected line after pod appeared")
	}
}

func TestNilClusterSubscribeDoesNotFail(t *testing.T) {
	mgr := New(nil, nil)
	defer mgr.Close()
	snap, lines, stop := mgr.Subscribe("demo")
	defer stop()
	if snap == nil {
		t.Fatal("snapshot is nil")
	}
	select {
	case _, ok := <-lines:
		if !ok {
			t.Fatal("channel closed")
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestContainerPrefersSandboxName(t *testing.T) {
	pod := testPod("demo")
	pod.Spec.Containers = []corev1.Container{
		{Name: "sidecar"},
		{Name: "sandbox"},
	}
	if got := containerName(pod); got != "sandbox" {
		t.Fatalf("container = %q", got)
	}
	pod.Spec.Containers = []corev1.Container{{Name: "only"}}
	if got := containerName(pod); got != "only" {
		t.Fatalf("container = %q", got)
	}
}

type openCall struct {
	namespace, pod, container string
	since                     *metav1.Time
}

type fakeCluster struct {
	mu      sync.Mutex
	ns      string
	sandbox *unstructured.Unstructured
	readers []io.ReadCloser
	opens   chan openCall
	openN   int
	last    openCall
	getErr  error
}

func newFakeCluster(rc io.ReadCloser) *fakeCluster {
	return &fakeCluster{
		ns:      "default",
		sandbox: sandboxObj("demo"),
		readers: []io.ReadCloser{rc},
		opens:   make(chan openCall, 8),
	}
}

func (f *fakeCluster) Namespace() string { return f.ns }

func (f *fakeCluster) GetSandbox(_ context.Context, name string) (*unstructured.Unstructured, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.sandbox == nil || f.sandbox.GetName() != name {
		return nil, errors.New("not found")
	}
	return f.sandbox.DeepCopy(), nil
}

func (f *fakeCluster) OpenLogs(_ context.Context, namespace, pod, container string, since *metav1.Time) (io.ReadCloser, error) {
	f.mu.Lock()
	f.openN++
	var rc io.ReadCloser
	if len(f.readers) > 0 {
		rc = f.readers[0]
		f.readers = f.readers[1:]
	}
	f.mu.Unlock()
	var sinceCopy *metav1.Time
	if since != nil {
		t := *since
		sinceCopy = &t
	}
	call := openCall{namespace: namespace, pod: pod, container: container, since: sinceCopy}
	f.mu.Lock()
	f.last = call
	f.mu.Unlock()
	select {
	case f.opens <- call:
	default:
	}
	if rc == nil {
		return nil, errors.New("no reader")
	}
	return rc, nil
}

func (f *fakeCluster) openCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.openN
}

func (f *fakeCluster) lastOpenMeta() openCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

func waitOpen(t *testing.T, f *fakeCluster) openCall {
	t.Helper()
	select {
	case call := <-f.opens:
		return call
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for OpenLogs")
		return openCall{}
	}
}

type fakeResolver struct {
	mu  sync.Mutex
	pod *corev1.Pod
}

func staticPod(name string) *fakeResolver {
	return &fakeResolver{pod: testPod(name)}
}

func (f *fakeResolver) set(pod *corev1.Pod) {
	f.mu.Lock()
	f.pod = pod
	f.mu.Unlock()
}

func (f *fakeResolver) Resolve(context.Context, *unstructured.Unstructured) (resources.Related, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return resources.Related{Pod: f.pod, Events: []models.TimelineEvent{}}, nil
}

func sandboxObj(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "agents.x-k8s.io/v1beta1",
		"kind":       "Sandbox",
		"metadata":   map[string]any{"name": name, "namespace": "default"},
	}}
}

func testPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "sandbox"}},
		},
	}
}

type notifyCloser struct {
	io.ReadCloser
	once   sync.Once
	closed chan struct{}
}

func (c *notifyCloser) Close() error {
	err := c.ReadCloser.Close()
	c.once.Do(func() { close(c.closed) })
	return err
}

func recvLine(t *testing.T, ch <-chan models.LogLine) models.LogLine {
	t.Helper()
	select {
	case line, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		return line
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for line")
		return models.LogLine{}
	}
}

func TestParseLineCRLF(t *testing.T) {
	got := ParseLine("2026-08-27T20:01:02Z hi\r", 0)
	if got.Message != "hi" {
		t.Fatalf("message = %q", got.Message)
	}
}

func TestSubscribeEmptyName(t *testing.T) {
	mgr := New(nil, nil)
	defer mgr.Close()
	snap, lines, stop := mgr.Subscribe("")
	defer stop()
	if snap == nil {
		t.Fatal("snapshot is nil")
	}
	if _, ok := <-lines; ok {
		t.Fatal("expected closed channel")
	}
}

func TestBufferCapsAt2000(t *testing.T) {
	pr, pw := io.Pipe()
	cluster := newFakeCluster(pr)
	mgr := newManager(cluster, staticPod("demo"))
	defer mgr.Close()
	_, live, stop := mgr.Subscribe("demo")
	defer stop()
	go func() {
		for range live {
		}
	}()
	waitOpen(t, cluster)

	go func() {
		for i := 0; i < bufferSize+5; i++ {
			if _, err := io.WriteString(pw, "2026-08-27T20:01:02Z x\n"); err != nil {
				return
			}
		}
	}()

	deadline := time.After(5 * time.Second)
	for {
		snap, _, stop2 := mgr.Subscribe("demo")
		n := len(snap)
		stop2()
		if n == bufferSize {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("buffer = %d, want %d", n, bufferSize)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
}
