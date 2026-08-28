package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateInputValidate(t *testing.T) {
	valid := CreateInput{
		Name:   "research-agent",
		Image:  "python:3.12-slim",
		CPU:    "500m",
		Memory: "1Gi",
	}
	tests := []struct {
		name string
		in   CreateInput
		want *Error
	}{
		{name: "ok", in: valid},
		{name: "single char name", in: CreateInput{Name: "a", Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}},
		{name: "max length name", in: CreateInput{Name: strings.Repeat("a", 63), Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}},
		{name: "cpu cores", in: CreateInput{Name: valid.Name, Image: valid.Image, CPU: "1", Memory: valid.Memory}},
		{name: "empty name", in: CreateInput{Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "uppercase name", in: CreateInput{Name: "My_Sandbox", Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "underscore name", in: CreateInput{Name: "research_agent", Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "leading hyphen", in: CreateInput{Name: "-agent", Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "trailing hyphen", in: CreateInput{Name: "agent-", Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "too long", in: CreateInput{Name: strings.Repeat("a", 64), Image: valid.Image, CPU: valid.CPU, Memory: valid.Memory}, want: ErrInvalidName},
		{name: "missing image", in: CreateInput{Name: valid.Name, CPU: valid.CPU, Memory: valid.Memory}, want: ErrImageRequired},
		{name: "whitespace image", in: CreateInput{Name: valid.Name, Image: "  ", CPU: valid.CPU, Memory: valid.Memory}, want: ErrImageRequired},
		{name: "missing cpu", in: CreateInput{Name: valid.Name, Image: valid.Image, Memory: valid.Memory}, want: ErrCPURequired},
		{name: "invalid cpu", in: CreateInput{Name: valid.Name, Image: valid.Image, CPU: "fast", Memory: valid.Memory}, want: ErrCPUInvalid},
		{name: "missing memory", in: CreateInput{Name: valid.Name, Image: valid.Image, CPU: valid.CPU}, want: ErrMemoryRequired},
		{name: "invalid memory", in: CreateInput{Name: valid.Name, Image: valid.Image, CPU: valid.CPU, Memory: "lots"}, want: ErrMemoryInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			err := in.Validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			var got *Error
			if !AsError(err, &got) {
				t.Fatalf("Validate() = %v (%T), want %v", err, err, tt.want)
			}
			if got.Kind != tt.want.Kind || got.Message != tt.want.Message {
				t.Fatalf("Validate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCreateInputValidateTrims(t *testing.T) {
	in := CreateInput{
		Name:   "  research-agent  ",
		Image:  "  python:3.12-slim  ",
		CPU:    " 500m ",
		Memory: " 1Gi ",
	}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	if in.Name != "research-agent" || in.Image != "python:3.12-slim" || in.CPU != "500m" || in.Memory != "1Gi" {
		t.Fatalf("trimmed = %+v", in)
	}
}

func TestCreateInputJSONTags(t *testing.T) {
	raw, err := json.Marshal(CreateInput{
		Name:              "research-agent",
		Image:             "python:3.12-slim",
		CPU:               "500m",
		Memory:            "1Gi",
		PersistentStorage: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "image", "cpu", "memory", "persistentStorage"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, raw)
		}
	}
	if got["persistentStorage"] != true {
		t.Fatalf("persistentStorage = %v", got["persistentStorage"])
	}
}

func TestSandboxJSONNullsAndEmptySlices(t *testing.T) {
	sb := Sandbox{
		Name:       "research-agent",
		Namespace:  "default",
		Status:     PhasePending,
		Image:      "python:3.12-slim",
		CPU:        "500m",
		Memory:     "1Gi",
		Node:       nil,
		IP:         nil,
		CreatedAt:  "2026-08-27T20:01:02Z",
		Conditions: []Condition{},
		Events:     []TimelineEvent{},
		YAML:       "apiVersion: agents.x-k8s.io/v1beta1\n",
	}
	raw, err := json.Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"name", "namespace", "status", "image", "cpu", "memory",
		"node", "ip", "createdAt", "persistentStorage", "conditions", "events", "yaml",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, raw)
		}
	}
	if got["node"] != nil || got["ip"] != nil {
		t.Fatalf("node/ip want null, got %s", raw)
	}
	if _, ok := got["conditions"].([]any); !ok {
		t.Fatalf("conditions want [], got %s", raw)
	}
	if _, ok := got["events"].([]any); !ok {
		t.Fatalf("events want [], got %s", raw)
	}
	if got["status"] != "Pending" {
		t.Fatalf("status = %v", got["status"])
	}

	node := "worker-1"
	ip := "10.0.0.8"
	sb.Node = &node
	sb.IP = &ip
	raw, err = json.Marshal(sb)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["node"] != "worker-1" || got["ip"] != "10.0.0.8" {
		t.Fatalf("node/ip = %s", raw)
	}
}

func TestConnectionJSONNulls(t *testing.T) {
	raw, err := json.Marshal(Connection{State: ConnectionError, Cluster: nil, Message: Ptr("load kubeconfig")})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["state"] != "error" || got["cluster"] != nil || got["message"] != "load kubeconfig" {
		t.Fatalf("got %s", raw)
	}
}

func TestNonNil(t *testing.T) {
	if got := NonNil([]Condition(nil)); got == nil || len(got) != 0 {
		t.Fatalf("NonNil(nil) = %v", got)
	}
	in := []Condition{{Type: "Ready"}}
	if got := NonNil(in); len(got) != 1 || got[0].Type != "Ready" {
		t.Fatalf("NonNil(in) = %v", got)
	}
}

func TestLifecycleErrorMessages(t *testing.T) {
	if got := ErrAlreadyExists.Format("demo"); got.Message != `Sandbox "demo" already exists.` {
		t.Fatalf("exists = %q", got.Message)
	}
	if got := ErrMissing.Format("demo"); got.Message != `Sandbox "demo" not found.` {
		t.Fatalf("missing = %q", got.Message)
	}
	if ErrPauseNotRunning.Message != "Only running sandboxes can be paused." {
		t.Fatalf("pause = %q", ErrPauseNotRunning.Message)
	}
	if ErrResumeNotPaused.Message != "Only paused sandboxes can be resumed." {
		t.Fatalf("resume = %q", ErrResumeNotPaused.Message)
	}
}
