package sandbox

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

const (
	sandboxAPIVersion = "agents.x-k8s.io/v1beta1"
	sandboxKind       = "Sandbox"
	sandboxContainer  = "sandbox"
	managedByLabel    = "app.kubernetes.io/managed-by"
	managedByValue    = "agent-sandbox-desktop"
	workspacePVCName  = "workspace"
	workspaceStorage  = "10Gi"
)

func FromCreate(ns string, in models.CreateInput) *unstructured.Unstructured {
	spec := map[string]any{
		"operatingMode": "Running",
		"service":       true,
		"podTemplate": map[string]any{
			"spec": map[string]any{
				"containers": []any{
					map[string]any{
						"name":  sandboxContainer,
						"image": in.Image,
						"resources": map[string]any{
							"requests": map[string]any{
								"cpu":    in.CPU,
								"memory": in.Memory,
							},
						},
					},
				},
			},
		},
	}
	if in.PersistentStorage {
		spec["volumeClaimTemplates"] = []any{
			map[string]any{
				"metadata": map[string]any{
					"name": workspacePVCName,
				},
				"spec": map[string]any{
					"accessModes": []any{"ReadWriteOnce"},
					"resources": map[string]any{
						"requests": map[string]any{
							"storage": workspaceStorage,
						},
					},
				},
			},
		}
	}

	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": sandboxAPIVersion,
		"kind":       sandboxKind,
		"metadata": map[string]any{
			"name":      in.Name,
			"namespace": ns,
			"labels": map[string]any{
				managedByLabel: managedByValue,
			},
		},
		"spec": spec,
	}}
}
