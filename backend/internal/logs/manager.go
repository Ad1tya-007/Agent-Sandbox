package logs

import (
	"bufio"
	"context"
	"io"
	"log"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/resources"
)

const (
	bufferSize       = 2000
	subBuffer        = 256
	minBackoff       = 500 * time.Millisecond
	maxBackoff       = 10 * time.Second
	sandboxContainer = "sandbox"
	maxScanToken     = 1 << 20
)

type Cluster interface {
	Namespace() string
	GetSandbox(ctx context.Context, name string) (*unstructured.Unstructured, error)
	OpenLogs(ctx context.Context, namespace, pod, container string, since *metav1.Time) (io.ReadCloser, error)
}

type Resolver interface {
	Resolve(ctx context.Context, obj *unstructured.Unstructured) (resources.Related, error)
}

type Manager struct {
	cluster Cluster
	mapper  Resolver

	mu      sync.Mutex
	streams map[string]*stream
	closed  bool
}

type stream struct {
	name string
	mgr  *Manager
	ctx  context.Context
	stop context.CancelFunc

	mu     sync.Mutex
	buffer []models.LogLine
	subs   map[uint64]chan models.LogLine
	nextID uint64
	offset int
	dead   bool
}

func New(k8s *kubernetes.Client, mapper *resources.Mapper) *Manager {
	var r Resolver
	if mapper != nil {
		r = mapper
	}
	return newManager(k8s, r)
}

func newManager(cluster Cluster, mapper Resolver) *Manager {
	return &Manager{
		cluster: cluster,
		mapper:  mapper,
		streams: make(map[string]*stream),
	}
}

func (m *Manager) Subscribe(name string) (snapshot []models.LogLine, lines <-chan models.LogLine, stop func()) {
	if name == "" {
		ch := make(chan models.LogLine)
		close(ch)
		return models.NonNil([]models.LogLine(nil)), ch, func() {}
	}
	for {
		s := m.getOrCreate(name)
		if s == nil {
			ch := make(chan models.LogLine)
			close(ch)
			return models.NonNil([]models.LogLine(nil)), ch, func() {}
		}
		snap, ch, unsub, ok := s.subscribe()
		if ok {
			return snap, ch, unsub
		}
	}
}

func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	streams := m.streams
	m.streams = nil
	m.mu.Unlock()
	for _, s := range streams {
		s.shutdown()
	}
}

func (m *Manager) getOrCreate(name string) *stream {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	if s, ok := m.streams[name]; ok {
		return s
	}
	ctx, cancel := context.WithCancel(context.Background())
	s := &stream{
		name: name,
		mgr:  m,
		ctx:  ctx,
		stop: cancel,
		subs: make(map[uint64]chan models.LogLine),
	}
	m.streams[name] = s
	go s.run()
	return s
}

func (m *Manager) drop(name string, s *stream) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.streams != nil && m.streams[name] == s {
		delete(m.streams, name)
	}
}

func (s *stream) subscribe() (snapshot []models.LogLine, lines <-chan models.LogLine, stop func(), ok bool) {
	s.mu.Lock()
	if s.dead {
		s.mu.Unlock()
		return nil, nil, nil, false
	}
	id := s.nextID
	s.nextID++
	ch := make(chan models.LogLine, subBuffer)
	s.subs[id] = ch
	snapshot = models.NonNil(append([]models.LogLine(nil), s.buffer...))
	s.mu.Unlock()

	var once sync.Once
	return snapshot, ch, func() {
		once.Do(func() { s.unsubscribe(id) })
	}, true
}

func (s *stream) unsubscribe(id uint64) {
	s.mu.Lock()
	ch, ok := s.subs[id]
	if ok {
		delete(s.subs, id)
		close(ch)
	}
	empty := len(s.subs) == 0
	if empty {
		s.dead = true
	}
	cancel := s.stop
	s.mu.Unlock()
	if !empty {
		return
	}
	if cancel != nil {
		cancel()
	}
	s.mgr.drop(s.name, s)
}

func (s *stream) shutdown() {
	s.mu.Lock()
	s.dead = true
	for id, ch := range s.subs {
		close(ch)
		delete(s.subs, id)
	}
	cancel := s.stop
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *stream) run() {
	defer s.stop()
	if s.mgr.cluster == nil {
		<-s.ctx.Done()
		return
	}

	backoff := minBackoff
	var since *metav1.Time
	first := true

	for {
		if s.ctx.Err() != nil {
			return
		}
		pod, container := s.resolve()
		if s.ctx.Err() != nil {
			return
		}
		if pod == nil {
			if !sleepCtx(s.ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		var sinceArg *metav1.Time
		if !first {
			sinceArg = since
		}
		ns := pod.Namespace
		if ns == "" {
			ns = s.mgr.cluster.Namespace()
		}
		rc, err := s.mgr.cluster.OpenLogs(s.ctx, ns, pod.Name, container, sinceArg)
		if err != nil || rc == nil {
			if err != nil {
				log.Printf("logs: open %s/%s: %v", ns, pod.Name, err)
			}
			if !sleepCtx(s.ctx, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}

		first = false
		backoff = minBackoff
		last := s.consume(onceReadCloser(rc))
		if last != nil {
			since = last
		}
		if s.ctx.Err() != nil {
			return
		}
		if !sleepCtx(s.ctx, backoff) {
			return
		}
		backoff = nextBackoff(backoff)
	}
}

func (s *stream) resolve() (*corev1.Pod, string) {
	if s.mgr.mapper == nil {
		return nil, ""
	}
	obj, err := s.mgr.cluster.GetSandbox(s.ctx, s.name)
	if err != nil || obj == nil {
		return nil, ""
	}
	related, err := s.mgr.mapper.Resolve(s.ctx, obj)
	if err != nil || related.Pod == nil {
		return nil, ""
	}
	return related.Pod, containerName(related.Pod)
}

func (s *stream) consume(rc io.ReadCloser) *metav1.Time {
	defer rc.Close()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-s.ctx.Done():
			_ = rc.Close()
		case <-done:
		}
	}()

	scanner := bufio.NewScanner(rc)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScanToken)
	var last *metav1.Time
	for scanner.Scan() {
		if s.ctx.Err() != nil {
			break
		}
		s.mu.Lock()
		line := ParseLine(scanner.Text(), s.offset)
		s.offset++
		s.appendLocked(line)
		s.fanoutLocked(line)
		s.mu.Unlock()
		if t, ok := parseTime(line.TS); ok {
			mt := metav1.NewTime(t)
			last = &mt
		}
	}
	return last
}

func (s *stream) appendLocked(line models.LogLine) {
	s.buffer = append(s.buffer, line)
	if len(s.buffer) > bufferSize {
		copy(s.buffer, s.buffer[len(s.buffer)-bufferSize:])
		s.buffer = s.buffer[:bufferSize]
	}
}

func (s *stream) fanoutLocked(line models.LogLine) {
	for _, ch := range s.subs {
		select {
		case ch <- line:
		default:
		}
	}
}

func containerName(pod *corev1.Pod) string {
	if pod == nil || len(pod.Spec.Containers) == 0 {
		return sandboxContainer
	}
	for _, c := range pod.Spec.Containers {
		if c.Name == sandboxContainer {
			return c.Name
		}
	}
	return pod.Spec.Containers[0].Name
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(d time.Duration) time.Duration {
	d *= 2
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

type onceCloser struct {
	io.ReadCloser
	once sync.Once
}

func onceReadCloser(rc io.ReadCloser) io.ReadCloser {
	if rc == nil {
		return nil
	}
	return &onceCloser{ReadCloser: rc}
}

func (c *onceCloser) Close() error {
	var err error
	c.once.Do(func() { err = c.ReadCloser.Close() })
	return err
}

var _ Cluster = (*kubernetes.Client)(nil)
var _ Resolver = (*resources.Mapper)(nil)
