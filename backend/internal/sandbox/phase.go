package sandbox

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func Phase(obj *unstructured.Unstructured) models.SandboxPhase {
	if obj == nil {
		return models.PhasePending
	}
	if !obj.GetDeletionTimestamp().IsZero() {
		return models.PhaseTerminating
	}
	if isSuspended(obj) {
		return models.PhasePaused
	}

	conditions := conditions(obj)
	if conditionTrue(conditions, "Finished") && conditionReason(conditions, "Finished") == "PodFailed" {
		return models.PhaseFailed
	}
	if readyFalseReason(conditions) == "PodFailed" {
		return models.PhaseFailed
	}
	if conditionTrue(conditions, "Ready") {
		return models.PhaseRunning
	}
	return models.PhasePending
}

func isSuspended(obj *unstructured.Unstructured) bool {
	mode, found, _ := unstructured.NestedString(obj.Object, "spec", "operatingMode")
	if found && strings.EqualFold(mode, "Suspended") {
		return true
	}
	replicas, found, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	return found && replicas == 0
}

func conditions(obj *unstructured.Unstructured) []map[string]any {
	raw, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func conditionTrue(conds []map[string]any, typeName string) bool {
	for _, c := range conds {
		if str(c["type"]) == typeName && str(c["status"]) == "True" {
			return true
		}
	}
	return false
}

func conditionReason(conds []map[string]any, typeName string) string {
	for _, c := range conds {
		if str(c["type"]) == typeName {
			return str(c["reason"])
		}
	}
	return ""
}

func readyFalseReason(conds []map[string]any) string {
	for _, c := range conds {
		if str(c["type"]) == "Ready" && str(c["status"]) == "False" {
			return str(c["reason"])
		}
	}
	return ""
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
