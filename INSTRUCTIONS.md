# Agent Sandbox Desktop

> A native desktop application for managing and debugging Agent Sandbox workloads running on Kubernetes.

---

# Vision

This project is **not** another Kubernetes dashboard.

It is **not** another AI chat application.

It is **not** a Docker Desktop clone.

The goal is to build a **developer tool** that platform engineers would actually use to interact with Agent Sandbox without touching `kubectl`.

The application should prioritize:

- Excellent backend architecture
- Clean code organization
- Real-time updates
- Native desktop experience
- A polished user experience

The philosophy is:

> **Three features implemented exceptionally well are far better than fifteen mediocre ones.**

---

# Tech Stack

## Desktop

- Tauri
- React
- TypeScript
- Vite

## Backend

- Go
- Kubernetes Go Client (`client-go`)
- Agent Sandbox CRDs

## Communication

- WebSocket
- JSON APIs

---

# Overall Architecture

```
                React UI
                    │
          WebSocket / HTTP
                    │
              Go Backend
                    │
         Kubernetes client-go
                    │
        Agent Sandbox CRDs
                    │
             Kubernetes Cluster
```

The Go backend is responsible for all business logic.

The frontend should never communicate directly with Kubernetes.

---

# Core Features

There are **ONLY THREE** core features.

Everything else is out of scope for v1.

---

# Feature 1 — Sandbox Lifecycle Management

## Goal

Allow developers to manage sandboxes without using kubectl.

---

## Create Sandbox

User fills out:

- Name
- Container Image
- CPU
- Memory
- Persistent Storage (optional)

Backend should:

- Validate input
- Generate Sandbox manifest
- Create resource through Kubernetes API
- Return success/error

---

## List Sandboxes

Display:

- Name
- Status
- Image
- CPU
- Memory
- Age
- Node

Requirements:

- Automatically update
- No refresh button
- Real-time synchronization

Use Kubernetes Watches.

Do NOT poll every few seconds.

---

## Pause / Resume

Allow pausing and resuming a sandbox.

The UI should immediately reflect changes through Kubernetes events.

---

## Delete Sandbox

Delete sandbox.

Show confirmation dialog.

Automatically remove from UI when deletion event arrives.

---

# Feature 2 — Live Sandbox Inspection

Clicking a sandbox opens a detailed inspector.

Display:

- Status
- Node
- IP
- Creation Time
- CPU Request
- Memory Request

This page contains only THREE tabs.

---

## Logs

Requirements:

- Stream logs live
- Auto-scroll
- Pause scrolling
- Search logs
- Copy logs

Implementation:

- Kubernetes Log API
- WebSocket streaming

No polling.

---

## Events

Instead of displaying raw Kubernetes events, translate them into a readable timeline.

Example:

```
Sandbox Created

↓

Container Started

↓

Image Pulled

↓

Running

↓

Restarted
```

The goal is readability.

---

## YAML

Display:

- Sandbox YAML
- Metadata
- Status
- Conditions

Read-only.

Power users should always be able to inspect the underlying resource.

---

# Feature 3 — Resource Explorer

This feature visualizes how Kubernetes resources are connected.

Selecting a sandbox should display something like:

```
Sandbox

├── Pod

├── PersistentVolumeClaim

├── Service

└── Events
```

Each resource can be selected to inspect details.

The purpose is educational.

Users should understand how Agent Sandbox maps onto Kubernetes objects.

---

# Backend Architecture

Backend code should be cleanly separated.

```
backend/

    cmd/

    internal/

        api/

        kubernetes/

        sandbox/

        watcher/

        websocket/

        logs/

        resources/

        models/
```

Responsibilities should never overlap.

---

## Kubernetes Layer

Responsible only for interacting with Kubernetes.

Responsibilities:

- Create Sandbox
- Delete Sandbox
- Watch Sandbox events
- Retrieve Pods
- Retrieve Services
- Retrieve PVCs
- Stream Logs

Business logic should not exist here.

---

## Sandbox Service

Business logic.

Responsibilities:

- Create sandbox
- Pause sandbox
- Resume sandbox
- Delete sandbox
- Validation
- Error handling

Should never know how HTTP works.

Should never know how React works.

---

## Watch Manager

One of the most important components.

Responsibilities:

- Register Kubernetes watches
- Receive resource events
- Publish events internally
- Broadcast updates through WebSocket

Everything in the UI should react to these events.

Avoid polling.

---

## Log Stream Manager

Responsibilities:

- Open log stream
- Forward logs
- Handle reconnects
- Close streams cleanly
- Support multiple viewers

---

## Resource Mapper

Given a Sandbox:

Automatically discover:

- Pod
- Service
- PVC
- Events

using labels and OwnerReferences.

This component teaches Kubernetes ownership relationships.

---

# Frontend Architecture

React should be organized by features.

```
src/

    components/

    pages/

    hooks/

    services/

    websocket/

    models/

    layouts/
```

Avoid putting API calls inside components.

---

# UI Principles

The application should feel similar to professional developer tools like:

- Docker Desktop
- Lens
- Kubernetes Dashboard
- GitHub Desktop

Design goals:

- Minimal
- Fast
- Native
- Information dense
- No unnecessary animations

---

# Non-Goals

The following are intentionally excluded from v1.

- AI Chat
- AI Agent creation
- Authentication
- Multi-user support
- RBAC management
- Cluster provisioning
- Metrics dashboards
- Prometheus integration
- Terminal emulator
- File browser
- Resource editing
- Templates
- Marketplace
- Plugin system

These may be explored after the MVP.

---

# Learning Objectives

This project is primarily intended to learn modern backend infrastructure.

The project should provide hands-on experience with:

- Kubernetes API
- Kubernetes Watches
- Custom Resource Definitions (CRDs)
- Go concurrency
- Goroutines
- Channels
- WebSockets
- Event-driven architecture
- Resource ownership
- Long-running connections
- Backend service architecture
- Clean project organization

---

# Quality Standards

Every feature should satisfy the following:

## Reliability

- Proper error handling
- Retry where appropriate
- No crashes

---

## Maintainability

- Small functions
- Clear package boundaries
- Meaningful naming
- No duplicated logic

---

## Performance

- No polling
- Event-driven updates
- Efficient resource usage

---

## User Experience

The user should never wonder:

- Is it still loading?
- Did my request succeed?
- Is the data current?

The interface should always communicate state clearly.

---

# Stretch Goals (After MVP)

Only begin these once the MVP is polished.

Possible future improvements:

- Multi-cluster support
- Namespace switching
- Cluster metrics
- Sandbox templates
- Warm Pools
- Sandbox Claims
- Snapshot system
- Advanced search
- Resource filters
- Keyboard shortcuts
- Dark/light themes

---

# Success Criteria

The MVP is complete when a user can:

1. Launch the desktop application.
2. Connect to an existing Kubernetes cluster.
3. View all Agent Sandboxes in real time.
4. Create a new sandbox.
5. Pause or resume a sandbox.
6. Delete a sandbox.
7. Inspect logs live.
8. View lifecycle events.
9. Inspect raw YAML.
10. Explore the Kubernetes resources created by the sandbox.

If these three core features are implemented with a clean architecture and polished UX, the project should resemble a production-quality internal developer tool rather than a feature-heavy prototype.
