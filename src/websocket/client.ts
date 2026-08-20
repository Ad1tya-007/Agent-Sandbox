import { mockBackend } from "@/mock/backend";
import type { LogEvent, WatchEvent } from "@/websocket/types";

export function connect(): void {
  mockBackend.connect();
}

export function subscribeWatch(
  listener: (event: WatchEvent) => void,
): () => void {
  return mockBackend.subscribe(listener);
}

export function subscribeLogs(
  name: string,
  listener: (event: LogEvent) => void,
): () => void {
  return mockBackend.subscribeLogs(name, listener);
}
