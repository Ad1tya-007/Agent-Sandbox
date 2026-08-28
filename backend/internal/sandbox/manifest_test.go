package sandbox

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"
)

func TestFromCreateSpec(t *testing.T) {
	in := models.CreateInput{
		Name:   "research-agent",
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	}
	obj := FromCreate("default", in)
	if obj == nil {
		t.Fatal("FromCreate returned nil")
	}

	if obj.GetAPIVersion() != sandboxAPIVersion {
		t.Fatalf("apiVersion = %q", obj.GetAPIVersion())
	}
	if obj.GetKind() != sandboxKind {
		t.Fatalf("kind = %q", obj.GetKind())
	}
	if obj.GetName() != "research-agent" {
		t.Fatalf("name = %q", obj.GetName())
	}
	if obj.GetNamespace() != "default" {
		t.Fatalf("namespace = %q", obj.GetNamespace())
	}
	if obj.GetLabels()[managedByLabel] != managedByValue {
		t.Fatalf("labels = %v", obj.GetLabels())
	}

	mode, found, err := unstructured.NestedString(obj.Object, "spec", "operatingMode")
	if err != nil || !found || mode != "Running" {
		t.Fatalf("operatingMode = %q found=%v err=%v", mode, found, err)
	}
	if _, found, err = unstructured.NestedFieldNoCopy(obj.Object, "spec", "replicas"); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatal("replicas must not be set on v1beta1 creates")
	}
	service, found, err := unstructured.NestedBool(obj.Object, "spec", "service")
	if err != nil || !found || !service {
		t.Fatalf("service = %v found=%v err=%v", service, found, err)
	}

	containers, found, err := unstructured.NestedSlice(obj.Object, "spec", "podTemplate", "spec", "containers")
	if err != nil || !found || len(containers) != 1 {
		t.Fatalf("containers = %v found=%v err=%v", containers, found, err)
	}
	c, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("container type %T", containers[0])
	}
	if str(c["name"]) != sandboxContainer {
		t.Fatalf("container name = %q", c["name"])
	}
	if str(c["image"]) != "python:3.12-slim" {
		t.Fatalf("image = %q", c["image"])
	}
	cpu, found, err := unstructured.NestedString(c, "resources", "requests", "cpu")
	if err != nil || !found || cpu != "500m" {
		t.Fatalf("cpu = %q found=%v err=%v", cpu, found, err)
	}
	memory, found, err := unstructured.NestedString(c, "resources", "requests", "memory")
	if err != nil || !found || memory != "1Gi" {
		t.Fatalf("memory = %q found=%v err=%v", memory, found, err)
	}

	_, found, err = unstructured.NestedSlice(obj.Object, "spec", "volumeClaimTemplates")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("volumeClaimTemplates present without persistentStorage")
	}
}

func TestFromCreateVolumeClaimTemplates(t *testing.T) {
	in := models.CreateInput{
		Name:              "research-agent",
		Image:             "python:3.12-slim",
		CPU:               "500m",
		Memory:            "1Gi",
		PersistentStorage: true,
	}
	obj := FromCreate("agents", in)
	if obj.GetNamespace() != "agents" {
		t.Fatalf("namespace = %q", obj.GetNamespace())
	}

	claims, found, err := unstructured.NestedSlice(obj.Object, "spec", "volumeClaimTemplates")
	if err != nil || !found || len(claims) != 1 {
		t.Fatalf("volumeClaimTemplates = %v found=%v err=%v", claims, found, err)
	}
	claim, ok := claims[0].(map[string]any)
	if !ok {
		t.Fatalf("claim type %T", claims[0])
	}
	name, found, err := unstructured.NestedString(claim, "metadata", "name")
	if err != nil || !found || name != workspacePVCName {
		t.Fatalf("pvc name = %q found=%v err=%v", name, found, err)
	}
	modes, found, err := unstructured.NestedStringSlice(claim, "spec", "accessModes")
	if err != nil || !found || len(modes) != 1 || modes[0] != "ReadWriteOnce" {
		t.Fatalf("accessModes = %v found=%v err=%v", modes, found, err)
	}
	storage, found, err := unstructured.NestedString(claim, "spec", "resources", "requests", "storage")
	if err != nil || !found || storage != workspaceStorage {
		t.Fatalf("storage = %q found=%v err=%v", storage, found, err)
	}
}

func TestFromCreateJSONShape(t *testing.T) {
	in := models.CreateInput{
		Name:   "research-agent",
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	}
	raw, err := json.Marshal(FromCreate("default", in).Object)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["apiVersion"] != sandboxAPIVersion || doc["kind"] != sandboxKind {
		t.Fatalf("gvk = %s", raw)
	}
	meta, _ := doc["metadata"].(map[string]any)
	if meta["name"] != "research-agent" || meta["namespace"] != "default" {
		t.Fatalf("metadata = %v", meta)
	}
	spec, _ := doc["spec"].(map[string]any)
	if spec == nil {
		t.Fatalf("missing spec in %s", raw)
	}
	if _, ok := spec["replicas"]; ok {
		t.Fatalf("replicas must not be set on v1beta1 creates: %v", spec["replicas"])
	}
	if spec["operatingMode"] != "Running" {
		t.Fatalf("operatingMode = %v", spec["operatingMode"])
	}
	if spec["service"] != true {
		t.Fatalf("service = %v", spec["service"])
	}
	if _, ok := spec["volumeClaimTemplates"]; ok {
		t.Fatal("volumeClaimTemplates present without persistentStorage")
	}
	containers := jsonPath(spec, "podTemplate", "spec", "containers")
	list, _ := containers.([]any)
	if len(list) != 1 {
		t.Fatalf("containers = %v", containers)
	}
	c, _ := list[0].(map[string]any)
	if c["name"] != sandboxContainer || c["image"] != "python:3.12-slim" {
		t.Fatalf("container = %v", c)
	}
	requests, _ := jsonPath(c, "resources", "requests").(map[string]any)
	if requests["cpu"] != "500m" || requests["memory"] != "1Gi" {
		t.Fatalf("requests = %v", requests)
	}
}

func TestFromCreateJSONShapeWithPVC(t *testing.T) {
	in := models.CreateInput{
		Name:              "research-agent",
		Image:             "python:3.12-slim",
		CPU:               "500m",
		Memory:            "1Gi",
		PersistentStorage: true,
	}
	raw, err := json.Marshal(FromCreate("agents", in).Object)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	claims, _ := jsonPath(doc, "spec", "volumeClaimTemplates").([]any)
	if len(claims) != 1 {
		t.Fatalf("volumeClaimTemplates = %v in %s", claims, raw)
	}
	claim, _ := claims[0].(map[string]any)
	if jsonPath(claim, "metadata", "name") != workspacePVCName {
		t.Fatalf("pvc name = %v", claim)
	}
	if jsonPath(claim, "spec", "resources", "requests", "storage") != workspaceStorage {
		t.Fatalf("storage = %v", claim)
	}
}

func jsonPath(v any, keys ...string) any {
	cur := v
	for _, key := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}
