package resources

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/kubernetes"
	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

type Core interface {
	ListPods(ctx context.Context, namespace, selector string) (*corev1.PodList, error)
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)
	ListPVCs(ctx context.Context, namespace, selector string) (*corev1.PersistentVolumeClaimList, error)
	GetPVC(ctx context.Context, namespace, name string) (*corev1.PersistentVolumeClaim, error)
	ListServices(ctx context.Context, namespace, selector string) (*corev1.ServiceList, error)
	GetService(ctx context.Context, namespace, name string) (*corev1.Service, error)
	ListEvents(ctx context.Context, namespace, fieldSelector string) (*corev1.EventList, error)
}

type Related struct {
	Pod     *corev1.Pod
	PVCs    []corev1.PersistentVolumeClaim
	Service *corev1.Service
	Events  []models.TimelineEvent
}

type Mapper struct {
	k8s Core
}

func New(k8s Core) *Mapper {
	return &Mapper{k8s: k8s}
}

func (m *Mapper) Resolve(ctx context.Context, obj *unstructured.Unstructured) (Related, error) {
	related := Related{
		PVCs:   []corev1.PersistentVolumeClaim{},
		Events: []models.TimelineEvent{},
	}
	if m == nil || m.k8s == nil || obj == nil {
		return related, nil
	}

	ns := obj.GetNamespace()
	name := obj.GetName()
	uid := obj.GetUID()
	selector := statusSelector(obj)

	pod, err := m.findPod(ctx, ns, name, uid, selector)
	if err != nil {
		return related, err
	}
	related.Pod = pod

	pvcs, err := m.findPVCs(ctx, obj, ns, name, uid, selector)
	if err != nil {
		return related, err
	}
	related.PVCs = pvcs

	svc, err := m.findService(ctx, obj, ns, name, uid, selector)
	if err != nil {
		return related, err
	}
	related.Service = svc

	events, err := m.events(ctx, ns, name, pod)
	if err != nil {
		return related, err
	}
	related.Events = events
	return related, nil
}

func (m *Mapper) findPod(ctx context.Context, ns, name string, uid types.UID, selector string) (*corev1.Pod, error) {
	var items []corev1.Pod
	if selector != "" {
		list, err := m.k8s.ListPods(ctx, ns, selector)
		if err != nil {
			return nil, err
		}
		if list != nil {
			items = list.Items
		}
	}
	if pod := pickOwnedOrFirst(items, name, uid); pod != nil {
		return pod, nil
	}

	pod, err := m.k8s.GetPod(ctx, ns, name)
	if ignoreNotFound(err) != nil {
		return nil, err
	}
	if pod == nil {
		return nil, nil
	}
	if len(pod.OwnerReferences) == 0 || ownedBySandbox(pod, name, uid) {
		return pod, nil
	}
	return nil, nil
}

func (m *Mapper) findPVCs(ctx context.Context, obj *unstructured.Unstructured, ns, name string, uid types.UID, selector string) ([]corev1.PersistentVolumeClaim, error) {
	byName := map[string]corev1.PersistentVolumeClaim{}
	if selector != "" {
		list, err := m.k8s.ListPVCs(ctx, ns, selector)
		if err != nil {
			return nil, err
		}
		if list != nil {
			for _, pvc := range list.Items {
				if ownedBySandbox(&pvc, name, uid) {
					byName[pvc.Name] = pvc
				}
			}
			if len(byName) == 0 {
				for _, pvc := range list.Items {
					byName[pvc.Name] = pvc
				}
			}
		}
	}
	for _, claim := range volumeClaimNames(obj) {
		pvc, err := m.k8s.GetPVC(ctx, ns, claim)
		if ignoreNotFound(err) != nil {
			return nil, err
		}
		if pvc != nil {
			byName[pvc.Name] = *pvc
		}
	}
	out := make([]corev1.PersistentVolumeClaim, 0, len(byName))
	for _, pvc := range byName {
		out = append(out, pvc)
	}
	return out, nil
}

func (m *Mapper) findService(ctx context.Context, obj *unstructured.Unstructured, ns, name string, uid types.UID, selector string) (*corev1.Service, error) {
	if svcName := statusService(obj); svcName != "" {
		svc, err := m.k8s.GetService(ctx, ns, svcName)
		if ignoreNotFound(err) != nil {
			return nil, err
		}
		if svc != nil {
			return svc, nil
		}
	}
	if selector == "" {
		return nil, nil
	}
	list, err := m.k8s.ListServices(ctx, ns, selector)
	if err != nil {
		return nil, err
	}
	if list == nil {
		return nil, nil
	}
	return pickOwnedOrFirstService(list.Items, name, uid), nil
}

func (m *Mapper) events(ctx context.Context, ns, sandboxName string, pod *corev1.Pod) ([]models.TimelineEvent, error) {
	var raw []corev1.Event
	names := []string{sandboxName}
	if pod != nil && pod.Name != sandboxName && pod.Name != "" {
		names = append(names, pod.Name)
	}
	for _, n := range names {
		if n == "" {
			continue
		}
		list, err := m.k8s.ListEvents(ctx, ns, "involvedObject.name="+n)
		if err != nil {
			return nil, err
		}
		if list != nil {
			raw = append(raw, list.Items...)
		}
	}
	return TranslateEvents(raw), nil
}

func pickOwnedOrFirst(pods []corev1.Pod, sandboxName string, uid types.UID) *corev1.Pod {
	var first *corev1.Pod
	for i := range pods {
		p := &pods[i]
		if ownedBySandbox(p, sandboxName, uid) {
			copy := *p
			return &copy
		}
		if first == nil {
			copy := *p
			first = &copy
		}
	}
	return first
}

func pickOwnedOrFirstService(svcs []corev1.Service, sandboxName string, uid types.UID) *corev1.Service {
	var first *corev1.Service
	for i := range svcs {
		s := &svcs[i]
		if ownedBySandbox(s, sandboxName, uid) {
			copy := *s
			return &copy
		}
		if first == nil {
			copy := *s
			first = &copy
		}
	}
	return first
}

func ownedBySandbox(obj metav1.Object, sandboxName string, uid types.UID) bool {
	if obj == nil {
		return false
	}
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind != kubernetes.SandboxKind {
			continue
		}
		if ref.APIVersion != "" {
			gv, err := schema.ParseGroupVersion(ref.APIVersion)
			if err != nil || gv.Group != kubernetes.SandboxGroup {
				continue
			}
		}
		if uid != "" && ref.UID == uid {
			return true
		}
		if sandboxName != "" && ref.Name == sandboxName {
			return true
		}
	}
	return false
}

func statusSelector(obj *unstructured.Unstructured) string {
	if obj == nil {
		return ""
	}
	if s, found, _ := unstructured.NestedString(obj.Object, "status", "selector"); found && s != "" {
		return s
	}
	if m, found, _ := unstructured.NestedStringMap(obj.Object, "status", "selector"); found && len(m) > 0 {
		return labels.Set(m).String()
	}
	if m, found, _ := unstructured.NestedStringMap(obj.Object, "status", "selector", "matchLabels"); found && len(m) > 0 {
		return labels.Set(m).String()
	}
	return ""
}

func statusService(obj *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(obj.Object, "status", "service")
	return s
}

func volumeClaimNames(obj *unstructured.Unstructured) []string {
	raw, found, _ := unstructured.NestedSlice(obj.Object, "spec", "volumeClaimTemplates")
	if !found {
		return nil
	}
	var names []string
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _, _ := unstructured.NestedString(m, "metadata", "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func ignoreNotFound(err error) error {
	if err == nil || errors.Is(err, kubernetes.ErrNotFound) {
		return nil
	}
	return err
}

var _ Core = (*kubernetes.Client)(nil)
