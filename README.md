# Agent Sandbox Desktop

Native desktop app for managing Agent Sandbox workloads on Kubernetes without `kubectl`.

This is a **developer tool**, not a cluster dashboard, AI chat app, or Docker Desktop clone. V1 is three features: sandbox lifecycle, live inspection (logs / events / YAML), and a resource explorer.

## Architecture

```
React UI  →  WebSocket / HTTP  →  Go backend  →  client-go  →  Agent Sandbox CRDs
```

Tauri (Rust) is the desktop shell only. Kubernetes access and product logic belong in Go. The UI never talks to the cluster.

The Go backend is not in this repo yet. The current tree is the desktop shell (Tauri 2 + React + TypeScript + Vite + shadcn/ui).

## Prerequisites

- [Rust](https://rustup.rs/) (stable, with `rustfmt` and `clippy`)
- Node 22+ (`nvm use` if you have nvm)
- A Kubernetes cluster (needed once the Go backend lands)

## Setup

```bash
npm install
cp .env.example .env
npm run tauri dev
```

## Checks

```bash
npm run check          # typecheck + Prettier + rustfmt + clippy + tests
npm run typecheck
npm run format:check
npm run rust:check
```

Pre-commit runs Prettier on staged files. Pre-push runs typecheck, Prettier, and rustfmt. Clippy and tests run in CI.

## Add UI components

```bash
npx shadcn@latest add dialog
```

## Build a release

```bash
npm run tauri build
```

## Recommended IDE setup

- [VS Code](https://code.visualstudio.com/) / Cursor + [Tauri](https://marketplace.visualstudio.com/items?itemName=tauri-apps.tauri-vscode) + [rust-analyzer](https://marketplace.visualstudio.com/items?itemName=rust-lang.rust-analyzer) + [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode)
