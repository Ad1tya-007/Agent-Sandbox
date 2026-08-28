package websocket

import "github.com/Ad1tya-007/Agent-Sandbox/backend/internal/models"

const (
	TypeConnection     = "connection"
	TypeSnapshot       = "snapshot"
	TypeSandboxAdded   = "sandbox.added"
	TypeSandboxUpdated = "sandbox.updated"
	TypeSandboxDeleted = "sandbox.deleted"
	TypeLine           = "line"
)

type ConnectionEvent struct {
	Type       string            `json:"type"`
	Connection models.Connection `json:"connection"`
}

type SnapshotEvent struct {
	Type      string           `json:"type"`
	Sandboxes []models.Sandbox `json:"sandboxes"`
}

type SandboxEvent struct {
	Type    string         `json:"type"`
	Sandbox models.Sandbox `json:"sandbox"`
}

type DeletedEvent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type LogSnapshotEvent struct {
	Type  string           `json:"type"`
	Lines []models.LogLine `json:"lines"`
}

type LogLineEvent struct {
	Type string         `json:"type"`
	Line models.LogLine `json:"line"`
}
