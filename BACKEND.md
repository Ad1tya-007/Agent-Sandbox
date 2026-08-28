# Backend Implementation Playbook

The React UI is done. It currently talks to an in-browser mock (`src/mock/backend.ts`). Your job is to replace that mock with a real Go backend that talks to Kubernetes and emits **the same JSON and WebSocket messages the UI already consumes**.

Do not redesign the product. Do not add a fourth feature. Do not put Kubernetes logic in Tauri/Rust or in React.

Work through the steps in order. Each step should leave the tree compiling (`go test ./...` from `backend/`). Do not skip ahead to HTTP until the domain types and Kubernetes adapter exist.

Canonical product spec: [`INSTRUCTIONS.md`](INSTRUCTIONS.md). Layer rules: `.cursor/rules/architecture.mdc`, `.cursor/rules/go-backend.mdc`.

---

## 0. What already exists (do not throw away)

The UI and a few Go stubs are already in the repo. **Extend them. Do not rewrite them.**

### UI (done — do not rebuild)

| Surface                          | How it gets data today                                                     |
| -------------------------------- | -------------------------------------------------------------------------- |
| Sandbox table + inspector header | `useSandboxes()` ← watch WebSocket                                         |
| Connection badge                 | `useConnection()` ← `{ type: "connection" }`                               |
| Create / pause / resume / delete | `src/services/sandboxes.ts` (HTTP commands)                                |
| Logs tab                         | `useSandboxLogs(name)` ← logs WebSocket                                    |
| Events tab                       | `sandbox.events` on the view model                                         |
| YAML tab                         | `sandbox.yaml` on the view model                                           |
| Resource explorer                | Derived in the UI from the Sandbox view model (`src/lib/resource-tree.ts`) |

The explorer does **not** call a separate REST tree API. The backend still needs a Resource Mapper internally: logs need the real Pod/container, and the Sandbox view model needs real node, IP, events, YAML, and whether a PVC exists.

### Go stubs (keep and complete)

```
backend/
  go.sum                              # checksums present; go.mod is MISSING — create it
  internal/
    api/sandboxes.go                  # routes + handler shapes; helpers missing
    models/errors.go                  # ErrorKind + constructors — keep as-is
    kubernetes/logs.go                # OpenLogs — needs Client
    sandbox/phase.go                  # Phase() from unstructured Sandbox — keep as-is
```

`api/sandboxes.go` already declares the HTTP contract and imports packages you must implement:

- `internal/sandbox` — `Service` with Create / Pause / Resume / Delete
- `internal/websocket` — `Hub`, `HandleWatch`, `HandleLogs`
- `internal/logs` — `Manager`
- `internal/models` — `CreateInput`, `CreateResult` (not written yet)

Module path already used by those imports:

```
github.com/Ad1tya-007/Agent-Sandbox/backend
```

### Target layout when you are done

```
backend/
  go.mod
  go.sum
  cmd/agent-sandbox/main.go
  internal/
    api/            # HTTP + JSON only (routes already sketched)
    websocket/      # accept, subscribe, broadcast
    sandbox/        # validation + lifecycle (phase.go already exists)
    kubernetes/     # client-go calls only (logs.go already exists)
    watcher/        # informers/watches → internal events
    logs/           # one upstream stream per sandbox, N subscribers
    resources/      # Sandbox → Pod / PVC / Service / Events
    models/         # view models + errors
```

Package import rules (fail the PR if broken):

- `sandbox` must not import `net/http` or UI types.
- `kubernetes` must not encode product policy (DNS-label rules, pause-only-when-Running, HTTP status codes).
- `api` decodes/encodes and maps `models.Error` → status codes. It does not create Sandbox manifests.
- `watcher` is the source of list/status/delete UI updates. No periodic `List` as the source of truth.

---

## 1. The contract the UI already expects

Match this byte-for-byte. Extra JSON fields are ignored by TypeScript; **missing or renamed fields break the UI**.

### Process

- Listen on `127.0.0.1:8787` (loopback only).
- Namespace: kubeconfig context namespace, override with `AGENT_SANDBOX_NAMESPACE`.
- Kubeconfig: `KUBECONFIG` or `~/.kube/config`.
- CORS: allow `http://localhost:1420` (Vite) so the desktop/dev UI can call you.

### HTTP

| Method   | Path                           | Request body  | Success                           |
| -------- | ------------------------------ | ------------- | --------------------------------- |
| `GET`    | `/healthz`                     | —             | `200` `{ "status": "ok" }`        |
| `POST`   | `/api/sandboxes`               | `CreateInput` | `200` `{ "name": "<dns-label>" }` |
| `POST`   | `/api/sandboxes/{name}/pause`  | —             | `204`                             |
| `POST`   | `/api/sandboxes/{name}/resume` | —             | `204`                             |
| `DELETE` | `/api/sandboxes/{name}`        | —             | `204`                             |
| `GET`    | `/ws`                          | —             | WebSocket (watch)                 |
| `GET`    | `/ws/logs?name=<sandbox>`      | —             | WebSocket (logs)                  |

These routes are already registered in `internal/api/sandboxes.go`. Implement the missing helpers (`decodeJSON`, `writeJSON`, `writeError`, `cors`) rather than changing the paths.

Create body (`models.CreateInput` / `CreateSandboxInput`):

```json
{
  "name": "research-agent",
  "image": "python:3.12-slim",
  "cpu": "500m",
  "memory": "1Gi",
  "persistentStorage": false
}
```

JSON names are **camelCase**. Use struct tags. Pause/resume/delete return empty bodies.

### Errors

Return JSON the frontend can put in a toast. Use this shape everywhere:

```json
{ "error": "Only running sandboxes can be paused." }
```

Map `models.Error.Kind` (already in `internal/models/errors.go`):

| Kind             | HTTP | When                                           |
| ---------------- | ---- | ---------------------------------------------- |
| `invalid`        | 400  | Bad name/image/cpu/memory, malformed JSON      |
| `not_found`      | 404  | Sandbox does not exist                         |
| `conflict`       | 409  | Name already exists                            |
| `conflict_state` | 409  | Pause when not Running, resume when not Paused |
| `internal`       | 500  | Unexpected Kubernetes / watch / log failures   |

Copy these user-facing strings (the mock already uses them; keep them stable):

- Name: `Name must be a DNS label: lowercase letters, numbers, and hyphens.`
- Image: `Container image is required.`
- CPU: `CPU request is required.`
- Memory: `Memory request is required.`
- Exists: `Sandbox "<name>" already exists.`
- Missing: `Sandbox "<name>" not found.`
- Pause: `Only running sandboxes can be paused.`
- Resume: `Only paused sandboxes can be resumed.`

Do not retry create/delete blindly. Do retry watch and log reconnects.

### Watch WebSocket (`GET /ws`)

One connection per UI. Messages are JSON objects with a `type` field.

```ts
type WatchEvent =
  | { type: "connection"; connection: Connection }
  | { type: "snapshot"; sandboxes: Sandbox[] }
  | { type: "sandbox.added"; sandbox: Sandbox }
  | { type: "sandbox.updated"; sandbox: Sandbox }
  | { type: "sandbox.deleted"; name: string };

type Connection = {
  state: "connecting" | "connected" | "disconnected" | "error";
  cluster: string | null;
  message: string | null;
};
```

On accept:

1. Send `{ type: "connection", connection: { state: "connecting", cluster: "<context>", message: "Connecting to cluster" } }`.
2. When the informer has synced, send `{ type: "connection", connection: { state: "connected", cluster: "<context>", message: "Watching sandboxes" } }`.
3. Immediately send `{ type: "snapshot", sandboxes: [...] }` (full current list, possibly empty).
4. After that, only diffs: `sandbox.added` / `sandbox.updated` / `sandbox.deleted`.

If kubeconfig is missing or the watch dies:

- `{ type: "connection", connection: { state: "error", cluster: null, message: "<why>" } }`
- Retry the watch with backoff. When it recovers, send `connected` then a fresh `snapshot` (not a stream of `added` for every object).

The UI has **no list HTTP endpoint and no refresh button**. If you do not push watch events, the table stays on skeletons forever.

### Logs WebSocket (`GET /ws/logs?name=<sandbox>`)

```ts
type LogEvent =
  | { type: "snapshot"; lines: LogLine[] }
  | { type: "line"; line: LogLine };

type LogLine = { id: string; ts: string; message: string };
```

On subscribe:

1. Send `{ type: "snapshot", lines: [...] }` (buffered tail, may be `[]`).
2. Then `{ type: "line", line }` for each new line.

`ts` is ISO-8601. `id` is unique per line (stable hash of `ts + message + offset` is fine). Kubernetes log API with `Timestamps: true` (already set in `kubernetes/logs.go`).

### Sandbox view model (every watch payload)

This is `src/models/sandbox.ts`. Populate every field.

```ts
type Sandbox = {
  name: string;
  namespace: string;
  status: "Pending" | "Running" | "Paused" | "Failed" | "Terminating";
  image: string;
  cpu: string;
  memory: string;
  node: string | null;
  ip: string | null;
  createdAt: string; // RFC3339
  persistentStorage: boolean;
  conditions: {
    type: string;
    status: "True" | "False" | "Unknown";
    message: string;
  }[];
  events: { id: string; title: string; detail: string; at: string }[];
  yaml: string; // read-only dump of the live object
};
```

`status` is a **product phase**, not a raw CRD field. `internal/sandbox/phase.go` already implements this from an unstructured Sandbox. Use it. Do not invent a second mapper.

Phase rules (already coded — do not change the meaning):

1. DeletionTimestamp set → `Terminating`
2. `spec.operatingMode == Suspended` **or** `spec.replicas == 0` → `Paused`
3. Ready=False reason `PodFailed`, or Finished=True reason `PodFailed` → `Failed`
4. Ready=True → `Running`
5. Else → `Pending`

Pause is **intent** (`operatingMode` / `replicas`). Observed readiness is the Ready condition. A just-resumed sandbox is `Pending` until Ready becomes True. That is correct.

### What you are not building

No auth, RBAC UI, metrics, terminal, file browser, YAML editing, templates, Warm Pools, Claims, multi-cluster, namespace picker. v1alpha1 Sandbox objects only.

---

## 2. Agent Sandbox CRD facts (use these, do not guess)

GVR:

```
Group:    agents.x-k8s.io
Version:  v1alpha1          // UI YAML tab also uses this
Resource: sandboxes
Kind:     Sandbox
```

Use the **dynamic client** + `unstructured.Unstructured`. Do not vendor the CRD Go types. The CRD is in flux (`replicas` vs `operatingMode`); unstructured plus the existing `Phase()` helper keeps you compatible.

Create body you apply (unstructured). Always set `spec.service: true` so the controller creates a headless Service the explorer can show.

```yaml
apiVersion: agents.x-k8s.io/v1alpha1
kind: Sandbox
metadata:
  name: research-agent
  namespace: default
  labels:
    app.kubernetes.io/managed-by: agent-sandbox-desktop
spec:
  replicas: 1
  operatingMode: Running
  service: true
  podTemplate:
    spec:
      containers:
        - name: sandbox
          image: python:3.12-slim
          resources:
            requests:
              cpu: "500m"
              memory: "1Gi"
  # only when persistentStorage is true:
  volumeClaimTemplates:
    - metadata:
        name: workspace
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
```

Set **both** `replicas: 1` and `operatingMode: Running` on create. Clusters that only understand one field will ignore the other.

Pause patch (try in this order, first one the API server accepts):

1. `{ spec: { operatingMode: "Suspended" } }`
2. `{ spec: { replicas: 0 } }`

Resume:

1. `{ spec: { operatingMode: "Running" } }`
2. `{ spec: { replicas: 1 } }`

Use merge patch. Read the object first so Pause/Resume can enforce product rules (only Running → pause, only Paused → resume) **in the sandbox service**, not in the Kubernetes package.

Status fields you will read when present:

- `status.conditions[]` — `type`, `status`, `reason`, `message`
- `status.podIPs[]` — first entry → view-model `ip`
- `status.nodeName` — view-model `node` (v1beta1; may be absent on v1alpha1)
- `status.service` — Service name
- `status.selector` — label selector for the backing Pod

If `nodeName` / `podIPs` are empty, fill `node` / `ip` from the owned Pod (`spec.nodeName`, `status.podIP`). Never invent worker names.

Ownership: the controller creates Pod, PVC, and Service with **OwnerReferences** pointing at the Sandbox. Discover children that way (and via `status.selector` / `status.service`). Do not scan the whole cluster by guessing names.

Typical backing names (do not hard-code as the only lookup):

- Pod: often the Sandbox name (not `{name}-0` — that is a mock UI fallback)
- PVC: from `volumeClaimTemplates[].metadata.name`
- Service: `status.service`

---

## 3. Step-by-step implementation

### Step 1 — Module, entrypoint, config

Create `backend/go.mod`:

```
module github.com/Ad1tya-007/Agent-Sandbox/backend

go 1.23
```

Pin Kubernetes libraries to the versions already in `go.sum` (`k8s.io/client-go v0.32.9`, `k8s.io/api`, `k8s.io/apimachinery`, `github.com/gorilla/websocket v1.5.3`, `sigs.k8s.io/yaml v1.4.0`). Run `go mod tidy` from `backend/`.

`cmd/agent-sandbox/main.go` (process entry only):

1. `signal.NotifyContext` for SIGINT/SIGTERM.
2. Read env: `KUBECONFIG`, `AGENT_SANDBOX_NAMESPACE`, optional `AGENT_SANDBOX_ADDR` default `127.0.0.1:8787`.
3. Build Kubernetes clients (in-cluster config is not required for v1; kubeconfig is).
4. Construct: kubernetes client → sandbox service → resource mapper → watcher → log manager → websocket hub → `api.New(...).Handler()`.
5. `http.Server` with graceful `Shutdown` on context cancel. Also stop informers and close log streams.

If kubeconfig cannot be loaded, still bind HTTP: `/healthz` works, `/ws` sends `connection.state = "error"` with the reason. Do not crash. The UI must be able to show “cannot reach cluster”.

### Step 2 — Models

Add `internal/models/sandbox.go` (and tiny helpers if needed). Mirror the TypeScript types. JSON tags **must** be camelCase.

```go
type SandboxPhase string // Pending, Running, Paused, Failed, Terminating

type CreateInput struct {
    Name               string `json:"name"`
    Image              string `json:"image"`
    CPU                string `json:"cpu"`
    Memory             string `json:"memory"`
    PersistentStorage  bool   `json:"persistentStorage"`
}

type CreateResult struct {
    Name string `json:"name"`
}

type Sandbox struct { /* fields from section 1 */ }

type Connection struct {
    State   string  `json:"state"`
    Cluster *string `json:"cluster"`
    Message *string `json:"message"`
}

type LogLine struct {
    ID      string `json:"id"`
    TS      string `json:"ts"`
    Message string `json:"message"`
}
```

Use pointers for `node`, `ip`, `cluster`, `message` so JSON `null` matches the UI.

Put watch/log envelope types next to the hub (`internal/websocket/events.go`) or in models. Discriminator field: `type`.

`CreateInput.Validate()` lives on the input / in `sandbox`, not in `api`:

- Name: Kubernetes DNS label, `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, length 1–63.
- Image, CPU, memory: non-empty after trim.
- Optionally parse CPU/memory with `resource.MustParse` / `resource.ParseQuantity` and return `invalid` on failure.

Unit-test validation. Do not hit the cluster.

### Step 3 — Kubernetes adapter (`internal/kubernetes`)

Thin. No product errors except wrapping transport failures as `error`.

`Client` holds:

- `kubernetes.Interface` (core: Pods, Services, PVCs, Events, Pod logs)
- `dynamic.Interface` (Sandbox CR)
- REST config (for later log streams — `logs.go` already uses `c.kube`)
- Cached GVR and namespace

Methods (names can vary; responsibilities cannot):

| Method                                               | Behavior                                                                                         |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| `CreateSandbox(ctx, obj *unstructured.Unstructured)` | `dynamic.Resource(gvr).Namespace(ns).Create`                                                     |
| `GetSandbox(ctx, name)`                              | Get                                                                                              |
| `DeleteSandbox(ctx, name)`                           | Delete                                                                                           |
| `PatchSandbox(ctx, name, patch []byte)`              | Merge patch                                                                                      |
| `SandboxGVR()`                                       | `schema.GroupVersionResource{Group:"agents.x-k8s.io", Version:"v1alpha1", Resource:"sandboxes"}` |
| `ListPods(ctx, ns, selector)`                        | CoreV1 Pods                                                                                      |
| `GetPod(ctx, ns, name)`                              |                                                                                                  |
| `ListPVCs(ctx, ns, selector)`                        |                                                                                                  |
| `ListServices(ctx, ns, selector)`                    |                                                                                                  |
| `ListEvents(ctx, ns, fieldSelector)`                 | Core Events for involvedObject                                                                   |
| `OpenLogs`                                           | **already written** — do not duplicate                                                           |

Also: `RESTConfig()`, `Namespace()`, `ClusterName()` (kubeconfig current context name).

Map `apierrors.IsNotFound` / `IsAlreadyExists` at the **service** layer into `models.NotFound` / `models.Conflict`, or provide a tiny `kubernetes.Translate(err)` that only classifies Kubernetes API errors — still no HTTP codes.

If v1alpha1 Sandbox is not installed, Get/List will 404 the CRD. Surface that as a connection error message like `Sandbox CRD not installed (agents.x-k8s.io/v1alpha1)`. Do not crash.

### Step 4 — Manifest builder (`internal/sandbox`)

`manifest.go`: `FromCreate(ns string, in models.CreateInput) *unstructured.Unstructured`.

Pure function. Unit-test the nested spec (image, requests, `service: true`, volumeClaimTemplates only when `persistentStorage`).

Keep `phase.go`. Add `project.go` (or `view.go`) that turns `*unstructured.Unstructured` plus optional related objects into `models.Sandbox`:

- `name`, `namespace`, `createdAt` from metadata
- `status` from `Phase(obj)`
- `image` / `cpu` / `memory` from first container in `spec.podTemplate.spec.containers`
- `persistentStorage` from non-empty `spec.volumeClaimTemplates`
- `conditions` from `status.conditions` (type, status, message)
- `node` / `ip` from status, else from the related Pod
- `yaml` from `sigs.k8s.io/yaml` after copying the object and deleting `metadata.managedFields`
- `events` from the resource mapper’s translated timeline (empty slice, never `null`)

### Step 5 — Sandbox service (`internal/sandbox/service.go`)

This is the only place that knows product rules.

```go
func (s *Service) Create(ctx, in) (*models.CreateResult, error)
func (s *Service) Pause(ctx, name) error
func (s *Service) Resume(ctx, name) error
func (s *Service) Delete(ctx, name) error
```

`Create`:

1. Trim + `Validate()`.
2. `FromCreate` → unstructured.
3. `k8s.CreateSandbox`.
4. On AlreadyExists → `models.Conflict`.
5. Return `{ Name: in.Name }`. **Do not wait for Ready.** The watch will push `sandbox.added` then `sandbox.updated`.

`Pause` / `Resume`:

1. Get Sandbox. Not found → `models.NotFound`.
2. Compute `Phase(obj)`.
3. Pause only if `Running`; resume only if `Paused`; else `models.ConflictState`.
4. Patch operatingMode then replicas as in section 2.
5. Return. UI flips when the watch sees the spec/status change.

`Delete`: Get for a clear 404, then Delete. The row stays until `sandbox.deleted` (the UI already works that way). While the object still exists with a deletionTimestamp, watches should emit `sandbox.updated` with `status: "Terminating"`.

Wire `var _ api.Sandboxes = (*sandbox.Service)(nil)` — already asserted in `sandboxes.go`.

### Step 6 — Resource mapper (`internal/resources`)

Given a Sandbox name/namespace/UID:

1. Pods in the namespace whose OwnerReference is this Sandbox (kind `Sandbox`, API `agents.x-k8s.io/v1alpha1`), or match `status.selector`.
2. PVCs with the same owner (or names from `volumeClaimTemplates`).
3. Service named `status.service`, else owned Service.
4. Events: field selector `involvedObject.name=<sandbox>` **and** events for the Pod name. Sort by last timestamp.

Do not List without a selector/owner filter.

Event translation (`translate.go`) — this is the Events tab. Map Kubernetes `reason` (and fallback `message`) to human titles. Deduplicate by `uid` or `name+timestamp`.

| Kubernetes reason (typical)             | Title                                      | Detail                                                              |
| --------------------------------------- | ------------------------------------------ | ------------------------------------------------------------------- |
| Created / SuccessfulCreate              | Sandbox Created                            | API server accepted the Sandbox resource / controller created child |
| Scheduled                               | Scheduled                                  | Assigned to `<node>`                                                |
| Pulling / Pulled                        | Image Pulled                               | `<image> pulled`                                                    |
| Created (container) / Started           | Container Started                          | sandbox container started                                           |
| Killing / BackOff / Unhealthy (restart) | Restarted                                  | Keep the original message as detail                                 |
| Failed / FailedScheduling               | Failed or keep Pending                     | Original message (e.g. insufficient CPU)                            |
| default                                 | `event.Reason` if non-empty else `"Event"` | `event.Message`                                                     |

`TimelineEvent.at` is RFC3339 from `lastTimestamp` or `eventTime`. `id` is the Event UID.

Unit-test translation with fake `corev1.Event` objects. No cluster.

The UI explorer still **guesses** child names (`{name}-0`, `{name}-workspace`) in `src/lib/resource-tree.ts`. Do not block MVP on rewriting that. Accurate node/IP/events/YAML on the Sandbox payload is the mapper’s v1 job. After the backend works, a small follow-up can pass real child names into the view model if you want the tree labels to match the cluster.

### Step 7 — Watch manager (`internal/watcher`)

Use a dynamic SharedInformer on the Sandbox GVR, namespace-scoped.

- `AddFunc` → publish `added`
- `UpdateFunc` → publish `updated` (skip if resourceVersion unchanged)
- `DeleteFunc` → publish `deleted` with the object name (handle `DeletedFinalStateUnknown`)

On each add/update, project to `models.Sandbox` using the mapper (pod + events). Keep projection cheap: use listers/informers for Pods and Events too if you need node/IP/events without extra List calls. Minimum viable: Sandbox informer + on-demand Get/List for children with a short cache.

Internal event bus: a channel or callback the Hub subscribes to. The watcher must not import `net/http` or gorilla.

`HasSynced` → tell the Hub to broadcast `connected` + `snapshot`.

On watch errors: log, set connection to `error` with a useful message, exponential backoff, restart. Never `os.Exit`.

Store the latest projected list in memory so a late WebSocket subscriber can get a snapshot without listing the API server.

### Step 8 — WebSocket hub (`internal/websocket`)

`Hub`:

- Register/unregister connections.
- Broadcast `WatchEvent` JSON to all watch subscribers.
- Latest `Connection` + latest sandbox list for new subscribers (replay `connection` then `snapshot`).

`HandleWatch`: gorilla upgrader, check origin (localhost / 127.0.0.1), register, read loop (discard client messages or handle ping), unregister on close.

`HandleLogs(manager)`: read `name` query; if empty, 400; else subscribe.

Write JSON with the same `type` strings as TypeScript (`sandbox.added`, not `Added`).

Concurrency: mutex around the subscriber maps. Never block the informer on a slow socket — per-connection buffered channel; drop or close if the buffer fills (document the choice; dropping is acceptable for a desktop tool, but then send a fresh snapshot on the next successful write if you can).

### Step 9 — Log stream manager (`internal/logs`)

One **upstream** Kubernetes follow-stream per sandbox name. N **subscribers** (inspectors).

Subscribe(name):

1. If no upstream: resolve Pod + container via mapper (first container, or name `sandbox`). If no pod yet, wait for it (watch/retry) and send empty snapshots until logs exist — do not fail the WebSocket.
2. `OpenLogs` with TailLines 200 the first time; on reconnect use `SinceTime`.
3. Scan lines. Kubernetes format with timestamps: `2026-08-27T20:01:02.123456789Z message`.
4. Buffer last ~2000 lines in memory for new subscribers (`snapshot`).
5. Fan out `line` events.

Unsubscribe: when the last subscriber leaves, cancel the context so `OpenLogs` closes (see `logs.go`). Do not leak goroutines.

Reconnect: if the stream errors and subscribers remain, backoff and reopen. Do not spin.

If the sandbox is Paused/Terminating and the pod is gone, send the buffered snapshot and idle until a pod exists again (resume).

### Step 10 — HTTP helpers and wiring (`internal/api`)

Complete `sandboxes.go` without changing route paths:

- `decodeJSON` — limit body size (~1MB), `json.Decoder.DisallowUnknownFields` optional, invalid → `models.Invalid`.
- `writeJSON` / `writeError` — map `models.Error` to the table in section 1; unknown errors → 500 `{ "error": "..." }` without panicking.
- `cors` — `OPTIONS` 204; allow the Vite origin; `Content-Type`, methods GET, POST, DELETE, OPTIONS.

`New` already takes `(sandboxes, hub, logMgr)`. `main` constructs those.

### Step 11 — Tests (write as you go, not as a dump at the end)

Minimum:

| Package                 | Tests                                                                                                                                                                        |
| ----------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `sandbox`               | Validate; `FromCreate` JSON shape; `Phase` (already have logic — table tests for Ready/Paused/Failed/Terminating); Pause/Resume state machine with a fake Kubernetes adapter |
| `resources`             | Event reason → title                                                                                                                                                         |
| `models`                | Error wrapping                                                                                                                                                               |
| `logs`                  | Line parse (`ts` + `message`); subscribe/unsubscribe closes upstream (fake `io.ReadCloser`)                                                                                  |
| `watcher` / `websocket` | Snapshot then added/updated/deleted; late subscriber gets snapshot                                                                                                           |

Fake the Kubernetes adapter with an interface defined **where the service needs it** (in `sandbox` or `kubernetes`). Do not require a cluster for `go test`.

### Step 12 — Flip the UI from mock to live (thin wiring only)

Do not rewrite components or hooks. Only the two service files plus env.

1. `src/services/sandboxes.ts` — `fetch` to `http://127.0.0.1:8787` (or `import.meta.env.VITE_API_BASE`). Parse `{ error }` into `ApiError` with `response.status`. Create returns `{ name }`. Pause/resume/delete expect 204.
2. `src/websocket/client.ts` — `ws://127.0.0.1:8787/ws` and `ws://127.0.0.1:8787/ws/logs?name=`. Parse JSON into `WatchEvent` / `LogEvent`. Reconnect on close with backoff; `useSandboxes` / `useConnection` already re-subscribe via `useEffect`.
3. Keep `VITE_USE_MOCK=true` as an escape hatch that still uses `src/mock/backend.ts` (README already documents this). Default **off** so `npm run desktop:dev` hits Go.
4. Vite `server.proxy` is optional; Tauri CSP already allows `http://127.0.0.1:*` and `ws://127.0.0.1:*`.

Do not add a Refresh button. Do not poll.

### Step 13 — Scripts and CI

- `npm run backend:dev` already runs `go run ./cmd/agent-sandbox` — that only works once `cmd/` exists; `scripts/dev.mjs` cwd is `backend/`.
- `package.json` `backend:check` currently does `gofmt` + `go test ./backend/...` from the **repo root**, which is the wrong module path. Fix it to run from `backend/` (e.g. `cd backend && test -z "$(gofmt -l .)" && go test ./...`).
- Add `npm run backend:check` to `npm run check`.
- Add a CI job (new workflow or extend `ci.yml`) that sets up Go 1.23 and runs `backend:check`. Frontend/Rust jobs stay as they are.

`.env.example` already documents `KUBECONFIG`. Add `AGENT_SANDBOX_NAMESPACE=` as a comment.

---

## 4. Suggested file checklist

Create/complete these. Tick them off in order.

```
backend/go.mod
backend/cmd/agent-sandbox/main.go

backend/internal/models/sandbox.go          # view models
backend/internal/models/errors.go           # EXISTS

backend/internal/kubernetes/client.go       # rest.Config, clients, namespace
backend/internal/kubernetes/sandbox.go      # create/get/delete/patch unstructured
backend/internal/kubernetes/core.go         # pods, pvcs, services, events
backend/internal/kubernetes/logs.go         # EXISTS

backend/internal/sandbox/phase.go           # EXISTS
backend/internal/sandbox/manifest.go
backend/internal/sandbox/project.go
backend/internal/sandbox/service.go
backend/internal/sandbox/*_test.go

backend/internal/resources/mapper.go
backend/internal/resources/events.go
backend/internal/resources/events_test.go

backend/internal/watcher/manager.go

backend/internal/websocket/hub.go
backend/internal/websocket/watch.go
backend/internal/websocket/logs.go

backend/internal/logs/manager.go
backend/internal/logs/parse.go

backend/internal/api/sandboxes.go           # EXISTS — add helpers in http.go
backend/internal/api/http.go                # decodeJSON, writeJSON, writeError, cors
```

---

## 5. Manual verification (MVP is not done until this works)

Against a cluster that has the Agent Sandbox CRDs and a working kubeconfig:

1. `cd backend && go run ./cmd/agent-sandbox` — process listens on `127.0.0.1:8787`.
2. `curl -sS http://127.0.0.1:8787/healthz` → `{"status":"ok"}`.
3. Open the UI (`npm run desktop:dev` or Vite + backend). Badge goes connecting → connected with the kube context name. Table hydrates (empty or real sandboxes). **No refresh.**
4. Create with the dialog (name, image, CPU, memory, optional PVC). HTTP 200 `{ "name" }`. Row appears via `sandbox.added`, then moves Pending → Running via `sandbox.updated`. Inspector shows node, IP, YAML of the live object.
5. Pause a Running sandbox → 204, then Paused via watch. Resume → Running (possibly through Pending). Wrong-state pause/resume → 409 toast, list unchanged.
6. Delete → confirmation in UI (already there) → 204 → row shows Terminating then disappears on `sandbox.deleted`.
7. Logs tab: snapshot then live lines, auto-scroll. Open the same sandbox in two windows if you can — both receive lines (multi-viewer). Close inspector — upstream cancels when the last subscriber leaves.
8. Events tab: human titles (Created → Image Pulled → Started → Running), not raw `Normal  Pulling  kubelet`.
9. YAML tab: real CR YAML, read-only.
10. Explorer: Sandbox / Pod / PVC (if requested) / Service / Events. Even with frontend-guessed child names, node/IP/storage on the inspector header must be real.

Failure cases to try once:

- Stop the API server / bad kubeconfig → connection `error`, process still up, `/healthz` still 200.
- Create duplicate name → 409.
- Invalid name `My_Sandbox` → 400.

---

## 6. Quality bar (reject your own PR if any of these fail)

- UI never talks to Kubernetes.
- No `setInterval` cluster refresh anywhere.
- Handlers do not call `dynamic.Create` directly.
- Kubernetes package does not return HTTP status codes.
- Watches and log streams reconnect; create/delete do not retry in a loop.
- Empty lists are `[]`, not `null`. Missing node/IP are `null`, not `""`.
- Functions stay small; projection/validation/HTTP stay in their packages.

When those hold and the ten success criteria in `INSTRUCTIONS.md` work against a real cluster, the backend is done. Stretch goals (multi-cluster, themes, Claims, Warm Pools) stay off the table.
