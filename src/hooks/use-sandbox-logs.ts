import { useEffect, useState } from "react";
import type { LogLine } from "@/models/sandbox";
import { subscribeLogs } from "@/websocket/client";

export function useSandboxLogs(name: string | null): {
  lines: LogLine[];
  live: boolean;
} {
  const [lines, setLines] = useState<LogLine[]>([]);

  useEffect(() => {
    if (!name) {
      setLines([]);
      return;
    }
    setLines([]);
    return subscribeLogs(name, (event) => {
      if (event.type === "snapshot") {
        setLines(event.lines);
        return;
      }
      setLines((current) => {
        const next = [...current, event.line];
        return next.length > 2000 ? next.slice(next.length - 2000) : next;
      });
    });
  }, [name]);

  return { lines, live: name != null };
}
