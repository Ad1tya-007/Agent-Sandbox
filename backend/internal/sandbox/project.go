package sandbox

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

type Related struct {
	Pod    *corev1.Pod
	Events []models.TimelineEvent
}

func Project(obj *unstructured.Unstructured, related Related) models.Sandbox {
	events := models.NonNil(related.Events)
	if obj == nil {
		return models.Sandbox{
			Status:     models.PhasePending,
			Conditions: []models.Condition{},
			Events:     events,
		}
	}

	image, cpu, memory := firstContainerResources(obj)
	node, ip := nodeAndIP(obj, related.Pod)
	return models.Sandbox{
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Status:            Phase(obj),
		Image:             image,
		CPU:               cpu,
		Memory:            memory,
		Node:              node,
		IP:                ip,
		CreatedAt:         createdAt(obj),
		PersistentStorage: hasVolumeClaims(obj),
		Conditions:        projectConditions(obj),
		Events:            events,
		YAML:              sandboxYAML(obj),
	}
}

func firstContainerResources(obj *unstructured.Unstructured) (image, cpu, memory string) {
	containers, found, _ := unstructured.NestedSlice(obj.Object, "spec", "podTemplate", "spec", "containers")
	if !found || len(containers) == 0 {
		return "", "", ""
	}
	m, ok := containers[0].(map[string]any)
	if !ok {
		return "", "", ""
	}
	image = str(m["image"])
	cpu, _, _ = unstructured.NestedString(m, "resources", "requests", "cpu")
	memory, _, _ = unstructured.NestedString(m, "resources", "requests", "memory")
	return image, cpu, memory
}

func hasVolumeClaims(obj *unstructured.Unstructured) bool {
	claims, found, _ := unstructured.NestedSlice(obj.Object, "spec", "volumeClaimTemplates")
	return found && len(claims) > 0
}

func projectConditions(obj *unstructured.Unstructured) []models.Condition {
	raw := conditions(obj)
	out := make([]models.Condition, 0, len(raw))
	for _, c := range raw {
		out = append(out, models.Condition{
			Type:    str(c["type"]),
			Status:  conditionStatus(str(c["status"])),
			Message: str(c["message"]),
		})
	}
	return models.NonNil(out)
}

func conditionStatus(s string) models.ConditionStatus {
	switch models.ConditionStatus(s) {
	case models.ConditionTrue, models.ConditionFalse, models.ConditionUnknown:
		return models.ConditionStatus(s)
	default:
		return models.ConditionUnknown
	}
}

func nodeAndIP(obj *unstructured.Unstructured, pod *corev1.Pod) (node, ip *string) {
	nodeName, _, _ := unstructured.NestedString(obj.Object, "status", "nodeName")
	ipAddr := firstPodIP(obj)
	if nodeName == "" && pod != nil {
		nodeName = pod.Spec.NodeName
	}
	if ipAddr == "" && pod != nil {
		ipAddr = pod.Status.PodIP
	}
	return models.NonEmptyPtr(nodeName), models.NonEmptyPtr(ipAddr)
}

func firstPodIP(obj *unstructured.Unstructured) string {
	raw, found, _ := unstructured.NestedSlice(obj.Object, "status", "podIPs")
	if found && len(raw) > 0 {
		switch v := raw[0].(type) {
		case string:
			if v != "" {
				return v
			}
		case map[string]any:
			if s := str(v["ip"]); s != "" {
				return s
			}
			if s := str(v["IP"]); s != "" {
				return s
			}
		}
	}
	s, _, _ := unstructured.NestedString(obj.Object, "status", "podIP")
	return s
}

func createdAt(obj *unstructured.Unstructured) string {
	ts := obj.GetCreationTimestamp()
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func sandboxYAML(obj *unstructured.Unstructured) string {
	clone := obj.DeepCopy()
	if clone.Object == nil {
		return ""
	}
	unstructured.RemoveNestedField(clone.Object, "metadata", "managedFields")
	raw, err := yaml.Marshal(clone.Object)
	if err != nil {
		return ""
	}
	return string(raw)
}
