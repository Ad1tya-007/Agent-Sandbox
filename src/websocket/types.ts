import type { Connection } from "@/models/connection";
import type { LogLine, Sandbox } from "@/models/sandbox";

export type WatchEvent =
  | { type: "connection"; connection: Connection }
  | { type: "snapshot"; sandboxes: Sandbox[] }
  | { type: "sandbox.added"; sandbox: Sandbox }
  | { type: "sandbox.updated"; sandbox: Sandbox }
  | { type: "sandbox.deleted"; name: string };

export type LogEvent =
  | { type: "snapshot"; lines: LogLine[] }
  | { type: "line"; line: LogLine };
