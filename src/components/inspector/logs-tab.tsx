import { useEffect, useMemo, useRef, useState } from "react";
import { Copy, Pause, Play } from "lucide-react";
import { toast } from "sonner";
import type { LogLine } from "@/models/sandbox";
import { formatClock } from "@/lib/format";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function LogsTab({ lines }: { lines: LogLine[] }) {
  const [query, setQuery] = useState("");
  const [paused, setPaused] = useState(false);
  const scroller = useRef<HTMLDivElement>(null);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return lines;
    return lines.filter((line) => line.message.toLowerCase().includes(q));
  }, [lines, query]);

  useEffect(() => {
    if (paused) return;
    const el = scroller.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [visible, paused]);

  async function copy() {
    const text = visible
      .map((line) => `${formatClock(line.ts)} ${line.message}`)
      .join("\n");
    await navigator.clipboard.writeText(text || "");
    toast.success("Logs copied");
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b px-3 py-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search logs"
          className="h-7 max-w-xs"
        />
        <div className="ml-auto flex items-center gap-1">
          <Button
            type="button"
            variant={paused ? "secondary" : "ghost"}
            size="xs"
            onClick={() => setPaused((value) => !value)}
          >
            {paused ? <Play /> : <Pause />}
            {paused ? "Resume scroll" : "Pause scroll"}
          </Button>
          <Button type="button" variant="ghost" size="xs" onClick={copy}>
            <Copy />
            Copy
          </Button>
        </div>
      </div>
      <div
        ref={scroller}
        className="min-h-0 flex-1 overflow-auto bg-[#0c0c0c] font-mono text-[12px] leading-5 text-neutral-200"
      >
        {visible.length === 0 ? (
          <p className="p-3 text-neutral-500">
            {lines.length === 0
              ? "Waiting for log stream…"
              : "No matching lines"}
          </p>
        ) : (
          <pre className="p-3 whitespace-pre-wrap">
            {visible.map((line) => (
              <div key={line.id} className="flex gap-3">
                <span className="text-neutral-500 shrink-0 tabular-nums">
                  {formatClock(line.ts)}
                </span>
                <span className={cn(query && "text-neutral-100")}>
                  {line.message}
                </span>
              </div>
            ))}
          </pre>
        )}
      </div>
    </div>
  );
}
