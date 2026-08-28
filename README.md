# Agent Sandbox Desktop

Native desktop app for managing Agent Sandbox workloads on Kubernetes without `kubectl`.

This is a **developer tool**, not a cluster dashboard, AI chat app, or Docker Desktop clone. V1 is three features: sandbox lifecycle, live inspection (logs / events / YAML), and a resource explorer.

## Architecture

```
React UI  →  WebSocket / HTTP  →  Go backend  →  client-go  →  Agent Sandbox CRDs
```

Tauri (Rust) is the desktop shell only. Kubernetes access and product logic belong in Go. The UI never talks to the cluster.

Backend implementation steps: [`BACKEND.md`](BACKEND.md).

## Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [Rust](https://rustup.rs/) (stable, with `rustfmt` and `clippy`)
- Node 22+ (`nvm use` if you have nvm)
- A Kubernetes cluster with the [Agent Sandbox](https://github.com/kubernetes-sigs/agent-sandbox) CRDs installed
- A kubeconfig (defaults to `~/.kube/config`, or set `KUBECONFIG`)

## Setup

```bash
npm install
cp .env.example .env
npm run tauri dev
```

`tauri dev` starts the Go backend on `127.0.0.1:8787` and the Vite UI. The backend watches Sandbox objects in the current kubeconfig context namespace (override with `AGENT_SANDBOX_NAMESPACE`).

To run the UI against the in-browser mock instead of a cluster:

```bash
VITE_USE_MOCK=true npm run dev
```

## Checks

```bash
npm run check          # typecheck + Prettier + rustfmt + clippy + tests + Go
npm run typecheck
npm run format:check
npm run rust:check
npm run backend:check
```

Pre-commit runs Prettier on staged files. Pre-push runs typecheck, Prettier, and rustfmt. Clippy, tests, and Go checks run in CI.

## Add UI components

```bash
npx shadcn@latest add dialog
```

## Build a release

```bash
npm run tauri build
```

## Recommended IDE setup

- [VS Code](https://code.visualstudio.com/) / Cursor + [Tauri](https://marketplace.visualstudio.com/items?itemName=tauri-apps.tauri-vscode) + [rust-analyzer](https://marketplace.visualstudio.com/items?itemName=rust-analyzer.rust-analyzer) + [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode)
