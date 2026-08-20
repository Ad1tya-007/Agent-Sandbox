import { sandboxToYaml } from "@/lib/sandbox-yaml";
import type { LogLine, Sandbox, TimelineEvent } from "@/models/sandbox";

function at(msAgo: number): string {
  return new Date(Date.now() - msAgo).toISOString();
}

function event(
  id: string,
  title: string,
  detail: string,
  msAgo: number,
): TimelineEvent {
  return { id, title, detail, at: at(msAgo) };
}

function line(id: string, message: string, msAgo: number): LogLine {
  return { id, ts: at(msAgo), message };
}

function withYaml(sandbox: Omit<Sandbox, "yaml">): Sandbox {
  const next = { ...sandbox, yaml: "" };
  next.yaml = sandboxToYaml(next);
  return next;
}

export type SeedRecord = {
  sandbox: Sandbox;
  logs: LogLine[];
};

export function createSeed(): SeedRecord[] {
  const research = withYaml({
    name: "research-agent",
    namespace: "default",
    status: "Running",
    image: "python:3.12-slim",
    cpu: "500m",
    memory: "1Gi",
    node: "worker-a",
    ip: "10.42.0.18",
    createdAt: at(2 * 60 * 60 * 1000),
    persistentStorage: true,
    conditions: [
      { type: "Ready", status: "True", message: "Sandbox is ready" },
      { type: "PodScheduled", status: "True", message: "Assigned to worker-a" },
    ],
    events: [
      event(
        "e1",
        "Sandbox Created",
        "API server accepted the Sandbox resource",
        2 * 60 * 60 * 1000,
      ),
      event(
        "e2",
        "Image Pulled",
        "python:3.12-slim pulled on worker-a",
        2 * 60 * 60 * 1000 - 20_000,
      ),
      event(
        "e3",
        "Container Started",
        "sandbox container started",
        2 * 60 * 60 * 1000 - 35_000,
      ),
      event(
        "e4",
        "Running",
        "Sandbox reported Ready",
        2 * 60 * 60 * 1000 - 40_000,
      ),
    ],
  });

  const evalRunner = withYaml({
    name: "eval-runner",
    namespace: "default",
    status: "Paused",
    image: "golang:1.23",
    cpu: "1",
    memory: "2Gi",
    node: "worker-b",
    ip: "10.42.1.7",
    createdAt: at(26 * 60 * 60 * 1000),
    persistentStorage: false,
    conditions: [
      { type: "Ready", status: "False", message: "Sandbox is paused" },
      { type: "PodScheduled", status: "True", message: "Assigned to worker-b" },
    ],
    events: [
      event(
        "e1",
        "Sandbox Created",
        "API server accepted the Sandbox resource",
        26 * 60 * 60 * 1000,
      ),
      event(
        "e2",
        "Image Pulled",
        "golang:1.23 pulled on worker-b",
        26 * 60 * 60 * 1000 - 15_000,
      ),
      event(
        "e3",
        "Container Started",
        "sandbox container started",
        26 * 60 * 60 * 1000 - 30_000,
      ),
      event(
        "e4",
        "Running",
        "Sandbox reported Ready",
        26 * 60 * 60 * 1000 - 40_000,
      ),
      event("e5", "Paused", "Workload paused by user", 40 * 60 * 1000),
    ],
  });

  const interpreter = withYaml({
    name: "code-interpreter",
    namespace: "default",
    status: "Pending",
    image: "ubuntu:24.04",
    cpu: "250m",
    memory: "512Mi",
    node: null,
    ip: null,
    createdAt: at(3 * 60 * 1000),
    persistentStorage: false,
    conditions: [
      { type: "Ready", status: "False", message: "Waiting for pod" },
      {
        type: "PodScheduled",
        status: "False",
        message: "0/3 nodes available: insufficient cpu",
      },
    ],
    events: [
      event(
        "e1",
        "Sandbox Created",
        "API server accepted the Sandbox resource",
        3 * 60 * 1000,
      ),
    ],
  });

  const nightly = withYaml({
    name: "nightly-bench",
    namespace: "default",
    status: "Failed",
    image: "nvidia/cuda:12.4.1-runtime-ubuntu22.04",
    cpu: "2",
    memory: "8Gi",
    node: "gpu-1",
    ip: "10.42.3.22",
    createdAt: at(6 * 60 * 60 * 1000),
    persistentStorage: true,
    conditions: [
      {
        type: "Ready",
        status: "False",
        message: "Container exited with code 1",
      },
      { type: "PodScheduled", status: "True", message: "Assigned to gpu-1" },
    ],
    events: [
      event(
        "e1",
        "Sandbox Created",
        "API server accepted the Sandbox resource",
        6 * 60 * 60 * 1000,
      ),
      event(
        "e2",
        "Image Pulled",
        "CUDA runtime image pulled on gpu-1",
        6 * 60 * 60 * 1000 - 45_000,
      ),
      event(
        "e3",
        "Container Started",
        "sandbox container started",
        6 * 60 * 60 * 1000 - 70_000,
      ),
      event(
        "e4",
        "Running",
        "Sandbox reported Ready",
        6 * 60 * 60 * 1000 - 80_000,
      ),
      event(
        "e5",
        "Restarted",
        "Container crashed (OOM) and was restarted",
        5 * 60 * 60 * 1000,
      ),
    ],
  });

  return [
    {
      sandbox: research,
      logs: [
        line("l1", "Starting sandbox runtime", 2 * 60 * 60 * 1000 - 40_000),
        line(
          "l2",
          "Mounted workspace PVC at /workspace",
          2 * 60 * 60 * 1000 - 39_000,
        ),
        line("l3", "Listening on :8080", 2 * 60 * 60 * 1000 - 38_000),
        line("l4", "Ready to accept sessions", 2 * 60 * 60 * 1000 - 37_000),
        line("l5", "session opened id=a1c3", 12 * 60 * 1000),
        line("l6", "ran: pytest tests/ -q", 11 * 60 * 1000),
        line("l7", "12 passed in 4.02s", 11 * 60 * 1000 - 4000),
      ],
    },
    {
      sandbox: evalRunner,
      logs: [
        line("l1", "Starting sandbox runtime", 26 * 60 * 60 * 1000 - 40_000),
        line("l2", "Listening on :8080", 26 * 60 * 60 * 1000 - 38_000),
        line("l3", "Received pause signal", 40 * 60 * 1000),
        line("l4", "Flushing in-flight work", 40 * 60 * 1000 - 1000),
        line("l5", "Sandbox paused", 40 * 60 * 1000 - 2000),
      ],
    },
    {
      sandbox: interpreter,
      logs: [],
    },
    {
      sandbox: nightly,
      logs: [
        line("l1", "Starting CUDA runtime", 6 * 60 * 60 * 1000 - 80_000),
        line("l2", "Detected GPU 0: NVIDIA L4", 6 * 60 * 60 * 1000 - 78_000),
        line("l3", "Allocating 8Gi workspace", 6 * 60 * 60 * 1000 - 76_000),
        line("l4", "fatal: CUDA out of memory", 5 * 60 * 60 * 1000 + 2000),
        line("l5", "container exited with code 1", 5 * 60 * 60 * 1000),
      ],
    },
  ];
}
