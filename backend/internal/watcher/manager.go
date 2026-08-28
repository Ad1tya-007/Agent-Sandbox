package watcher

import (
	"context"
	"errors"
	"io"
	"log"
	"sort"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/resources"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/sandbox"
)

const (
	subscriberBuffer = 32
	projectTimeout   = 3 * time.Second
	minBackoff       = 500 * time.Millisecond
	maxBackoff       = 30 * time.Second
)

type EventType string

const (
	EventAdded   EventType = "added"
	EventUpdated EventType = "updated"
	EventDeleted EventType = "deleted"
	EventSynced  EventType = "synced"
	EventError   EventType = "error"
)

// Event is the internal watch bus. The Hub maps these onto WebSocket messages.
type Event struct {
	Type    EventType
	Sandbox models.Sandbox
	Name    string
}

type Resolver interface {
	Resolve(ctx context.Context, obj *unstructured.Unstructured) (resources.Related, error)
}

type cluster interface {
	Err() error
	Namespace() string
	ClusterName() string
	Dynamic() dynamic.Interface
}

type Manager struct {
	k8s       cluster
	mapper    Resolver
	listWatch cache.ListerWatcher

	mu          sync.RWMutex
	items       map[string]models.Sandbox
	connection  models.Connection
	clusterName string
	subs        map[uint64]chan Event
	nextSub     uint64
	synced      bool
	started     bool
	stopped     bool
	cancel      context.CancelFunc
}

func New(k8s *kubernetes.Client, mapper *resources.Mapper) *Manager {
	var r Resolver
	if mapper != nil {
		r = mapper
	}
	return newManager(k8s, r)
}

func newManager(k8s cluster, mapper Resolver) *Manager {
	m := &Manager{
		k8s:    k8s,
		mapper: mapper,
		items:  make(map[string]models.Sandbox),
		subs:   make(map[uint64]chan Event),
	}
	if k8s != nil {
		m.clusterName = k8s.ClusterName()
	}
	m.connection = connecting(m.clusterName)
	return m
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	ctx, m.cancel = context.WithCancel(ctx)
	if m.k8s != nil {
		m.clusterName = m.k8s.ClusterName()
	}
	m.connection = connecting(m.clusterName)
	lw := m.listWatch
	m.mu.Unlock()

	if err := m.clientErr(); err != nil {
		m.markError(err)
		return
	}
	go m.run(ctx, lw)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

func (m *Manager) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, subscriberBuffer)
	m.mu.Lock()
	if m.stopped {
		m.mu.Unlock()
		return ch, func() {}
	}
	id := m.nextSub
	m.nextSub++
	m.subs[id] = ch
	m.mu.Unlock()
	return ch, func() {
		m.mu.Lock()
		delete(m.subs, id)
		m.mu.Unlock()
	}
}

func (m *Manager) Snapshot() []models.Sandbox {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]models.Sandbox, 0, len(m.items))
	for _, sb := range m.items {
		out = append(out, sb)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return models.NonNil(out)
}

func (m *Manager) Connection() models.Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connection
}

func (m *Manager) HasSynced() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.synced
}

func (m *Manager) run(ctx context.Context, lw cache.ListerWatcher) {
	backoff := minBackoff
	for {
		if lw == nil {
			lw = m.kubernetesListWatch(ctx)
		}
		ns := ""
		if m.k8s != nil {
			ns = m.k8s.Namespace()
		}
		log.Printf("watcher: watching sandboxes in namespace %q", ns)
		m.runInformer(ctx, lw)
		if ctx.Err() != nil {
			return
		}
		m.markError(errors.New("sandbox watch stopped"))
		log.Printf("watcher: restarting in %s", backoff)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
		lw = m.kubernetesListWatch(ctx)
	}
}

func (m *Manager) runInformer(ctx context.Context, lw cache.ListerWatcher) {
	informer := cache.NewSharedIndexInformer(lw, &unstructured.Unstructured{}, 0, cache.Indexers{})
	_ = informer.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
		m.onWatchError(ctx, err)
	})

	reg, err := informer.AddEventHandler(cache.ResourceEventHandlerDetailedFuncs{
		AddFunc:    m.handleAdd,
		UpdateFunc: m.handleUpdate,
		DeleteFunc: m.handleDelete,
	})
	if err != nil {
		m.markError(err)
		return
	}

	go func() {
		if cache.WaitForCacheSync(ctx.Done(), reg.HasSynced) {
			m.markSynced()
		}
	}()
	informer.Run(ctx.Done())
}

func (m *Manager) kubernetesListWatch(ctx context.Context) cache.ListerWatcher {
	if m.k8s == nil || m.k8s.Dynamic() == nil {
		return nil
	}
	nsi := m.k8s.Dynamic().Resource(kubernetes.SandboxGVR()).Namespace(m.k8s.Namespace())
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			list, err := nsi.List(ctx, options)
			return list, kubernetes.Translate(err)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			w, err := nsi.Watch(ctx, options)
			if err != nil {
				if ctx.Err() != nil {
					return nil, err
				}
				m.markError(kubernetes.Translate(err))
				return nil, err
			}
			m.maybeRecover()
			return w, nil
		},
	}
}

func (m *Manager) handleAdd(obj any, isInInitialList bool) {
	sb, ok := m.upsert(obj)
	if !ok {
		return
	}
	if isInInitialList || !m.HasSynced() {
		return
	}
	m.publish(Event{Type: EventAdded, Sandbox: sb})
}

func (m *Manager) handleUpdate(oldObj, newObj any) {
	m.maybeRecover()
	oldU := asUnstructured(oldObj)
	newU := asUnstructured(newObj)
	if oldU != nil && newU != nil && oldU.GetResourceVersion() != "" && oldU.GetResourceVersion() == newU.GetResourceVersion() {
		return
	}
	sb, ok := m.upsert(newObj)
	if !ok || !m.HasSynced() {
		return
	}
	m.publish(Event{Type: EventUpdated, Sandbox: sb})
}

func (m *Manager) handleDelete(obj any) {
	name := objectName(obj)
	if name == "" {
		return
	}
	m.mu.Lock()
	delete(m.items, name)
	synced := m.synced
	m.mu.Unlock()
	if synced {
		m.publish(Event{Type: EventDeleted, Name: name})
	}
}

func (m *Manager) upsert(obj any) (models.Sandbox, bool) {
	u := asUnstructured(obj)
	if u == nil || u.GetName() == "" {
		return models.Sandbox{}, false
	}
	sb := m.project(u)
	m.mu.Lock()
	m.items[sb.Name] = sb
	m.mu.Unlock()
	return sb, true
}

func (m *Manager) project(obj *unstructured.Unstructured) models.Sandbox {
	related := sandbox.Related{Events: []models.TimelineEvent{}}
	if m.mapper != nil {
		ctx, cancel := context.WithTimeout(context.Background(), projectTimeout)
		defer cancel()
		got, err := m.mapper.Resolve(ctx, obj)
		if err != nil {
			log.Printf("watcher: resolve %s: %v", obj.GetName(), err)
		} else {
			related = sandbox.Related{Pod: got.Pod, Events: models.NonNil(got.Events)}
		}
	}
	return sandbox.Project(obj, related)
}

func (m *Manager) markSynced() {
	m.mu.Lock()
	m.synced = true
	m.connection = connected(m.clusterName)
	m.mu.Unlock()
	log.Printf("watcher: synced, %d sandbox(es)", len(m.Snapshot()))
	m.publish(Event{Type: EventSynced})
}

func (m *Manager) markError(err error) {
	if err == nil || isBenignWatchError(err) {
		return
	}
	msg := watchMessage(err)
	m.mu.Lock()
	m.connection = models.Connection{
		State:   models.ConnectionError,
		Cluster: nil,
		Message: models.Ptr(msg),
	}
	m.mu.Unlock()
	log.Printf("watcher: %s", msg)
	m.publish(Event{Type: EventError})
}

func (m *Manager) maybeRecover() {
	m.mu.Lock()
	if m.connection.State != models.ConnectionError || !m.synced {
		m.mu.Unlock()
		return
	}
	m.connection = connected(m.clusterName)
	m.mu.Unlock()
	log.Printf("watcher: recovered, %d sandbox(es)", len(m.Snapshot()))
	m.publish(Event{Type: EventSynced})
}

func (m *Manager) onWatchError(ctx context.Context, err error) {
	if ctx.Err() != nil || isBenignWatchError(err) {
		return
	}
	m.markError(err)
}

func (m *Manager) publish(ev Event) {
	m.mu.RLock()
	subs := make([]chan Event, 0, len(m.subs))
	for _, ch := range m.subs {
		subs = append(subs, ch)
	}
	m.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (m *Manager) clientErr() error {
	if m.k8s == nil {
		return errors.New("kubernetes client is not configured")
	}
	if err := m.k8s.Err(); err != nil {
		return err
	}
	if m.k8s.Dynamic() == nil {
		return errors.New("kubernetes client is not configured")
	}
	return nil
}

func connecting(cluster string) models.Connection {
	return models.Connection{
		State:   models.ConnectionConnecting,
		Cluster: models.NonEmptyPtr(cluster),
		Message: models.Ptr("Connecting to cluster"),
	}
}

func connected(cluster string) models.Connection {
	return models.Connection{
		State:   models.ConnectionConnected,
		Cluster: models.NonEmptyPtr(cluster),
		Message: models.Ptr("Watching sandboxes"),
	}
}

func asUnstructured(obj any) *unstructured.Unstructured {
	switch t := obj.(type) {
	case *unstructured.Unstructured:
		return t
	case unstructured.Unstructured:
		return t.DeepCopy()
	case cache.DeletedFinalStateUnknown:
		return asUnstructured(t.Obj)
	default:
		return nil
	}
}

func objectName(obj any) string {
	if u := asUnstructured(obj); u != nil {
		return u.GetName()
	}
	if t, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		_, name, err := cache.SplitMetaNamespaceKey(t.Key)
		if err == nil {
			return name
		}
	}
	return ""
}

func isBenignWatchError(err error) bool {
	return err == nil ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		apierrors.IsResourceExpired(err) ||
		apierrors.IsGone(err)
}

func watchMessage(err error) string {
	translated := kubernetes.Translate(err)
	if errors.Is(translated, kubernetes.ErrCRDMissing) {
		return kubernetes.ErrCRDMissing.Error()
	}
	return translated.Error()
}
