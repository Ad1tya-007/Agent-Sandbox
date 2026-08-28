import { mockBackend } from "@/mock/backend";
import type { Connection } from "@/models/connection";
import type { Sandbox } from "@/models/sandbox";
import type { LogEvent, WatchEvent } from "@/websocket/types";

const useMock = import.meta.env.VITE_USE_MOCK === "true";
const apiBase = (
  import.meta.env.VITE_API_BASE ?? "http://127.0.0.1:8787"
).replace(/\/$/, "");
const wsBase = apiBase.replace(/^http/, "ws");

const INITIAL_BACKOFF_MS = 500;
const MAX_BACKOFF_MS = 8000;

type WatchListener = (event: WatchEvent) => void;
type LogListener = (event: LogEvent) => void;

function parseWatchEvent(data: string): WatchEvent | null {
  try {
    const value = JSON.parse(data) as WatchEvent;
    if (!value || typeof value !== "object" || typeof value.type !== "string") {
      return null;
    }
    return value;
  } catch {
    return null;
  }
}

function parseLogEvent(data: string): LogEvent | null {
  try {
    const value = JSON.parse(data) as LogEvent;
    if (!value || typeof value !== "object" || typeof value.type !== "string") {
      return null;
    }
    return value;
  } catch {
    return null;
  }
}

class LiveWatch {
  private ws: WebSocket | null = null;
  private readonly listeners = new Set<WatchListener>();
  private lastConnection: Connection | null = null;
  private sandboxes: Sandbox[] = [];
  private hydrated = false;
  private backoff = INITIAL_BACKOFF_MS;
  private reconnectTimer: number | null = null;
  private started = false;

  connect(): void {
    if (this.started) return;
    this.started = true;
    this.open();
  }

  subscribe(listener: WatchListener): () => void {
    this.listeners.add(listener);
    if (this.lastConnection) {
      listener({ type: "connection", connection: this.lastConnection });
    }
    if (this.hydrated) {
      listener({ type: "snapshot", sandboxes: this.sandboxes });
    }
    return () => {
      this.listeners.delete(listener);
    };
  }

  private apply(event: WatchEvent): void {
    switch (event.type) {
      case "connection":
        this.lastConnection = event.connection;
        break;
      case "snapshot":
        this.sandboxes = event.sandboxes;
        this.hydrated = true;
        break;
      case "sandbox.added":
        this.sandboxes = this.sandboxes.some(
          (item) => item.name === event.sandbox.name,
        )
          ? this.sandboxes.map((item) =>
              item.name === event.sandbox.name ? event.sandbox : item,
            )
          : [...this.sandboxes, event.sandbox];
        this.hydrated = true;
        break;
      case "sandbox.updated":
        this.sandboxes = this.sandboxes.map((item) =>
          item.name === event.sandbox.name ? event.sandbox : item,
        );
        break;
      case "sandbox.deleted":
        this.sandboxes = this.sandboxes.filter(
          (item) => item.name !== event.name,
        );
        break;
    }
  }

  private open(): void {
    if (this.reconnectTimer != null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }

    const connecting: Connection = {
      state: "connecting",
      cluster: this.lastConnection?.cluster ?? null,
      message: "Connecting to cluster",
    };
    this.lastConnection = connecting;
    this.emit({ type: "connection", connection: connecting });

    const ws = new WebSocket(`${wsBase}/ws`);
    this.ws = ws;

    ws.onopen = () => {
      this.backoff = INITIAL_BACKOFF_MS;
    };

    ws.onmessage = (event) => {
      const parsed = parseWatchEvent(String(event.data));
      if (!parsed) return;
      this.apply(parsed);
      this.emit(parsed);
    };

    ws.onclose = () => {
      if (this.ws === ws) this.ws = null;
      const connection: Connection = {
        state: "disconnected",
        cluster: this.lastConnection?.cluster ?? null,
        message: "Lost connection to backend",
      };
      this.lastConnection = connection;
      this.emit({ type: "connection", connection });
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer != null) return;
    const wait = this.backoff;
    this.backoff = Math.min(this.backoff * 2, MAX_BACKOFF_MS);
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, wait);
  }

  private emit(event: WatchEvent): void {
    for (const listener of this.listeners) listener(event);
  }
}

const liveWatch = new LiveWatch();

function subscribeLogsLive(name: string, listener: LogListener): () => void {
  let ws: WebSocket | null = null;
  let timer: number | null = null;
  let backoff = INITIAL_BACKOFF_MS;
  let stopped = false;

  const open = () => {
    if (stopped) return;
    const socket = new WebSocket(
      `${wsBase}/ws/logs?name=${encodeURIComponent(name)}`,
    );
    ws = socket;
    socket.onopen = () => {
      backoff = INITIAL_BACKOFF_MS;
    };
    socket.onmessage = (event) => {
      const parsed = parseLogEvent(String(event.data));
      if (parsed) listener(parsed);
    };
    socket.onclose = () => {
      if (stopped) return;
      timer = window.setTimeout(() => {
        timer = null;
        backoff = Math.min(backoff * 2, MAX_BACKOFF_MS);
        open();
      }, backoff);
    };
  };

  open();

  return () => {
    stopped = true;
    if (timer != null) window.clearTimeout(timer);
    ws?.close();
  };
}

export function connect(): void {
  if (useMock) {
    mockBackend.connect();
    return;
  }
  liveWatch.connect();
}

export function subscribeWatch(listener: WatchListener): () => void {
  if (useMock) return mockBackend.subscribe(listener);
  return liveWatch.subscribe(listener);
}

export function subscribeLogs(name: string, listener: LogListener): () => void {
  if (useMock) return mockBackend.subscribeLogs(name, listener);
  return subscribeLogsLive(name, listener);
}
