import { Pause, Play, Trash2 } from "lucide-react";
import type { Sandbox } from "@/models/sandbox";
import { formatTimestamp } from "@/lib/format";
import { StatusBadge } from "@/components/sandbox/status-badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-muted-foreground text-[11px] tracking-wide uppercase">
        {label}
      </p>
      <p className="truncate font-mono text-xs">{value}</p>
    </div>
  );
}

export function InspectorHeader({
  sandbox,
  busy,
  onPause,
  onResume,
  onDelete,
}: {
  sandbox: Sandbox;
  busy: boolean;
  onPause: () => void;
  onResume: () => void;
  onDelete: () => void;
}) {
  const terminating = sandbox.status === "Terminating";

  return (
    <div className="shrink-0">
      <div className="flex flex-wrap items-center gap-2 px-3 py-2">
        <h2 className="text-sm font-semibold">{sandbox.name}</h2>
        <StatusBadge status={sandbox.status} />
        <div className="ml-auto flex items-center gap-1">
          {sandbox.status === "Paused" ? (
            <Button
              variant="outline"
              size="xs"
              disabled={busy || terminating}
              onClick={onResume}
            >
              <Play />
              Resume
            </Button>
          ) : (
            <Button
              variant="outline"
              size="xs"
              disabled={busy || sandbox.status !== "Running"}
              onClick={onPause}
            >
              <Pause />
              Pause
            </Button>
          )}
          <Button
            variant="destructive"
            size="xs"
            disabled={busy || terminating}
            onClick={onDelete}
          >
            <Trash2 />
            Delete
          </Button>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-2 px-3 pb-3 sm:grid-cols-3 lg:grid-cols-6">
        <Stat label="Node" value={sandbox.node ?? "—"} />
        <Stat label="IP" value={sandbox.ip ?? "—"} />
        <Stat label="Created" value={formatTimestamp(sandbox.createdAt)} />
        <Stat label="CPU request" value={sandbox.cpu} />
        <Stat label="Memory request" value={sandbox.memory} />
        <Stat
          label="Storage"
          value={sandbox.persistentStorage ? "PVC attached" : "Ephemeral"}
        />
      </div>
      <Separator />
    </div>
  );
}
