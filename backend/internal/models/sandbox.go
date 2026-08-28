package models

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

type SandboxPhase string

const (
	PhasePending     SandboxPhase = "Pending"
	PhaseRunning     SandboxPhase = "Running"
	PhasePaused      SandboxPhase = "Paused"
	PhaseFailed      SandboxPhase = "Failed"
	PhaseTerminating SandboxPhase = "Terminating"
)

type ConditionStatus string

const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

type ConnectionState string

const (
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionConnected    ConnectionState = "connected"
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionError        ConnectionState = "error"
)

type Condition struct {
	Type    string          `json:"type"`
	Status  ConditionStatus `json:"status"`
	Message string          `json:"message"`
}

type TimelineEvent struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	At     string `json:"at"`
}

type Sandbox struct {
	Name              string          `json:"name"`
	Namespace         string          `json:"namespace"`
	Status            SandboxPhase    `json:"status"`
	Image             string          `json:"image"`
	CPU               string          `json:"cpu"`
	Memory            string          `json:"memory"`
	Node              *string         `json:"node"`
	IP                *string         `json:"ip"`
	CreatedAt         string          `json:"createdAt"`
	PersistentStorage bool            `json:"persistentStorage"`
	Conditions        []Condition     `json:"conditions"`
	Events            []TimelineEvent `json:"events"`
	YAML              string          `json:"yaml"`
}

type CreateInput struct {
	Name              string `json:"name"`
	Image             string `json:"image"`
	CPU               string `json:"cpu"`
	Memory            string `json:"memory"`
	PersistentStorage bool   `json:"persistentStorage"`
}

type CreateResult struct {
	Name string `json:"name"`
}

type Connection struct {
	State   ConnectionState `json:"state"`
	Cluster *string         `json:"cluster"`
	Message *string         `json:"message"`
}

type LogLine struct {
	ID      string `json:"id"`
	TS      string `json:"ts"`
	Message string `json:"message"`
}

var (
	ErrInvalidName     = Invalid("Name must be a DNS label: lowercase letters, numbers, and hyphens.")
	ErrImageRequired   = Invalid("Container image is required.")
	ErrCPURequired     = Invalid("CPU request is required.")
	ErrCPUInvalid      = Invalid("CPU request is invalid.")
	ErrMemoryRequired  = Invalid("Memory request is required.")
	ErrMemoryInvalid   = Invalid("Memory request is invalid.")
	ErrAlreadyExists   = Conflict(`Sandbox "%s" already exists.`)
	ErrMissing         = NotFound(`Sandbox "%s" not found.`)
	ErrPauseNotRunning = ConflictState("Only running sandboxes can be paused.")
	ErrResumeNotPaused = ConflictState("Only paused sandboxes can be resumed.")
)

var dnsLabel = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func Ptr[T any](v T) *T { return &v }

func NonEmptyPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func NonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (in *CreateInput) Validate() error {
	if in == nil {
		return ErrInvalidName
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Image = strings.TrimSpace(in.Image)
	in.CPU = strings.TrimSpace(in.CPU)
	in.Memory = strings.TrimSpace(in.Memory)

	if len(in.Name) < 1 || len(in.Name) > 63 || !dnsLabel.MatchString(in.Name) {
		return ErrInvalidName
	}
	if in.Image == "" {
		return ErrImageRequired
	}
	if in.CPU == "" {
		return ErrCPURequired
	}
	if _, err := resource.ParseQuantity(in.CPU); err != nil {
		return ErrCPUInvalid
	}
	if in.Memory == "" {
		return ErrMemoryRequired
	}
	if _, err := resource.ParseQuantity(in.Memory); err != nil {
		return ErrMemoryInvalid
	}
	return nil
}
