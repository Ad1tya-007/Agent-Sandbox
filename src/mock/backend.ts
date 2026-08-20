import { sandboxToYaml } from "@/lib/sandbox-yaml";
import type { Connection } from "@/models/connection";
import type {
  CreateSandboxInput,
  LogLine,
  Sandbox,
  TimelineEvent,
} from "@/models/sandbox";
import { createSeed } from "@/mock/seed";
import { ApiError } from "@/services/errors";
import type { LogEvent, WatchEvent } from "@/websocket/types";

const NAME_RE = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
const CLUSTER = "kind-agent-sandbox";

type WatchListener = (event: WatchEvent) => void;
type LogListener = (event: LogEvent) => void;

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function clone<T>(value: T): T {
  return structuredClone(value);
}

function nowIso(): string {
  return new Date().toISOString();
}

function nextId(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 9)}`;
}

function timeline(title: string, detail: string): TimelineEvent {
  return { id: nextId("evt"), title, detail, at: nowIso() };
}

function refreshYaml(sandbox: Sandbox): Sandbox {
  const next = { ...sandbox };
  next.yaml = sandboxToYaml(next);
  return next;
}

class MockBackend {
  private sandboxes = new Map<string, Sandbox>();
  private logs = new Map<string, LogLine[]>();
  private watchers = new Set<WatchListener>();
  private logWatchers = new Map<string, Set<LogListener>>();
  private logTimer: number | null = null;
  private logTick = 0;
  private started = false;
  private connection: Connection = {
    state: "disconnected",
    cluster: null,
    message: "Not connected",
  };

  connect(): void {
    if (this.started) return;
    this.started = true;
    this.setConnection({
      state: "connecting",
      cluster: CLUSTER,
      message: "Connecting to cluster",
    });

    window.setTimeout(() => {
      const seed = createSeed();
      for (const record of seed) {
        this.sandboxes.set(record.sandbox.name, record.sandbox);
        this.logs.set(record.sandbox.name, record.logs);
      }
      this.setConnection({
        state: "connected",
        cluster: CLUSTER,
        message: "Watching sandboxes",
      });
      this.emit({
        type: "snapshot",
        sandboxes: this.list(),
      });
      this.ensureLogPump();
    }, 450);
  }

  subscribe(listener: WatchListener): () => void {
    this.watchers.add(listener);
    listener({ type: "connection", connection: this.connection });
    if (this.connection.state === "connected") {
      listener({ type: "snapshot", sandboxes: this.list() });
    }
    return () => {
      this.watchers.delete(listener);
    };
  }

  subscribeLogs(name: string, listener: LogListener): () => void {
    let set = this.logWatchers.get(name);
    if (!set) {
      set = new Set();
      this.logWatchers.set(name, set);
    }
    set.add(listener);
    listener({ type: "snapshot", lines: clone(this.logs.get(name) ?? []) });
    this.ensureLogPump();
    return () => {
      set.delete(listener);
      if (set.size === 0) this.logWatchers.delete(name);
      this.ensureLogPump();
    };
  }

  async create(input: CreateSandboxInput): Promise<{ name: string }> {
    await delay(280);
    const name = input.name.trim();
    const image = input.image.trim();
    const cpu = input.cpu.trim();
    const memory = input.memory.trim();

    if (!NAME_RE.test(name) || name.length > 63) {
      throw new ApiError(
        "Name must be a DNS label: lowercase letters, numbers, and hyphens.",
      );
    }
    if (!image) throw new ApiError("Container image is required.");
    if (!cpu) throw new ApiError("CPU request is required.");
    if (!memory) throw new ApiError("Memory request is required.");
    if (this.sandboxes.has(name)) {
      throw new ApiError(`Sandbox "${name}" already exists.`, 409);
    }

    const createdAt = nowIso();
    const sandbox = refreshYaml({
      name,
      namespace: "default",
      status: "Pending",
      image,
      cpu,
      memory,
      node: null,
      ip: null,
      createdAt,
      persistentStorage: input.persistentStorage,
      conditions: [
        { type: "Ready", status: "False", message: "Waiting for pod" },
        { type: "PodScheduled", status: "Unknown", message: "Scheduling" },
      ],
      events: [
        timeline("Sandbox Created", "API server accepted the Sandbox resource"),
      ],
      yaml: "",
    });
    this.sandboxes.set(name, sandbox);
    this.logs.set(name, [
      { id: nextId("log"), ts: createdAt, message: "Creating sandbox" },
    ]);
    this.emit({ type: "sandbox.added", sandbox: clone(sandbox) });

    window.setTimeout(() => this.promoteCreated(name), 900);
    return { name };
  }

  async pause(name: string): Promise<void> {
    await delay(220);
    const sandbox = this.require(name);
    if (sandbox.status !== "Running") {
      throw new ApiError("Only running sandboxes can be paused.");
    }
    sandbox.status = "Paused";
    sandbox.conditions = [
      { type: "Ready", status: "False", message: "Sandbox is paused" },
      {
        type: "PodScheduled",
        status: "True",
        message: `Assigned to ${sandbox.node}`,
      },
    ];
    sandbox.events.push(timeline("Paused", "Workload paused by user"));
    this.logs.get(name)?.push({
      id: nextId("log"),
      ts: nowIso(),
      message: "Sandbox paused",
    });
    this.publishUpdate(sandbox);
  }

  async resume(name: string): Promise<void> {
    await delay(220);
    const sandbox = this.require(name);
    if (sandbox.status !== "Paused") {
      throw new ApiError("Only paused sandboxes can be resumed.");
    }
    sandbox.status = "Running";
    sandbox.conditions = [
      { type: "Ready", status: "True", message: "Sandbox is ready" },
      {
        type: "PodScheduled",
        status: "True",
        message: `Assigned to ${sandbox.node}`,
      },
    ];
    sandbox.events.push(timeline("Running", "Workload resumed by user"));
    this.logs.get(name)?.push({
      id: nextId("log"),
      ts: nowIso(),
      message: "Sandbox resumed",
    });
    this.publishUpdate(sandbox);
  }

  async remove(name: string): Promise<void> {
    await delay(220);
    const sandbox = this.require(name);
    sandbox.status = "Terminating";
    sandbox.conditions = [
      { type: "Ready", status: "False", message: "Sandbox is terminating" },
    ];
    sandbox.events.push(timeline("Terminating", "Delete requested"));
    this.publishUpdate(sandbox);
    window.setTimeout(() => {
      this.sandboxes.delete(name);
      this.logs.delete(name);
      this.logWatchers.delete(name);
      this.emit({ type: "sandbox.deleted", name });
    }, 650);
  }

  private promoteCreated(name: string): void {
    const sandbox = this.sandboxes.get(name);
    if (!sandbox || sandbox.status !== "Pending") return;
    const node = name.charCodeAt(0) % 2 === 0 ? "worker-a" : "worker-b";
    const octet = 20 + (this.sandboxes.size % 200);
    sandbox.node = node;
    sandbox.ip = `10.42.0.${octet}`;
    sandbox.status = "Running";
    sandbox.conditions = [
      { type: "Ready", status: "True", message: "Sandbox is ready" },
      { type: "PodScheduled", status: "True", message: `Assigned to ${node}` },
    ];
    sandbox.events.push(
      timeline("Image Pulled", `${sandbox.image} pulled on ${node}`),
      timeline("Container Started", "sandbox container started"),
      timeline("Running", "Sandbox reported Ready"),
    );
    this.logs.get(name)?.push(
      { id: nextId("log"), ts: nowIso(), message: `Pulled ${sandbox.image}` },
      { id: nextId("log"), ts: nowIso(), message: "Listening on :8080" },
      {
        id: nextId("log"),
        ts: nowIso(),
        message: "Ready to accept sessions",
      },
    );
    this.publishUpdate(sandbox);
  }

  private publishUpdate(sandbox: Sandbox): void {
    const next = refreshYaml(sandbox);
    this.sandboxes.set(next.name, next);
    this.emit({ type: "sandbox.updated", sandbox: clone(next) });
  }

  private require(name: string): Sandbox {
    const sandbox = this.sandboxes.get(name);
    if (!sandbox) throw new ApiError(`Sandbox "${name}" not found.`, 404);
    return sandbox;
  }

  private list(): Sandbox[] {
    return [...this.sandboxes.values()].map(clone);
  }

  private setConnection(connection: Connection): void {
    this.connection = connection;
    this.emit({ type: "connection", connection });
  }

  private emit(event: WatchEvent): void {
    for (const listener of this.watchers) listener(event);
  }

  private ensureLogPump(): void {
    const shouldRun =
      this.connection.state === "connected" && this.logWatchers.size > 0;
    if (shouldRun && this.logTimer == null) {
      this.logTimer = window.setInterval(() => this.pumpLogs(), 2200);
    }
    if (!shouldRun && this.logTimer != null) {
      window.clearInterval(this.logTimer);
      this.logTimer = null;
    }
  }

  private pumpLogs(): void {
    this.logTick += 1;
    for (const [name, listeners] of this.logWatchers) {
      const sandbox = this.sandboxes.get(name);
      if (!sandbox || sandbox.status !== "Running") continue;
      const line: LogLine = {
        id: nextId("log"),
        ts: nowIso(),
        message: this.logMessage(name, this.logTick),
      };
      const buffer = this.logs.get(name) ?? [];
      buffer.push(line);
      if (buffer.length > 2000) buffer.splice(0, buffer.length - 2000);
      this.logs.set(name, buffer);
      for (const listener of listeners) listener({ type: "line", line });
    }
  }

  private logMessage(name: string, tick: number): string {
    const samples = [
      "heartbeat ok",
      "session still active",
      "wrote checkpoint to /workspace",
      "gc: collected 2 objects",
      "accepted connection from 10.42.0.1",
    ];
    return `${name}: ${samples[tick % samples.length]}`;
  }
}

export const mockBackend = new MockBackend();
