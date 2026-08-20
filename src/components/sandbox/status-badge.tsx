import type { SandboxPhase } from "@/models/sandbox";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";

const STYLES: Record<SandboxPhase, string> = {
  Running:
    "border-transparent bg-emerald-500/12 text-emerald-700 dark:text-emerald-400",
  Pending:
    "border-transparent bg-amber-500/12 text-amber-800 dark:text-amber-400",
  Paused: "border-transparent bg-muted text-muted-foreground",
  Failed: "",
  Terminating:
    "border-transparent bg-orange-500/12 text-orange-800 dark:text-orange-400",
};

const DOT: Record<SandboxPhase, string> = {
  Running: "bg-emerald-500",
  Pending: "bg-amber-500",
  Paused: "bg-muted-foreground",
  Failed: "bg-destructive",
  Terminating: "bg-orange-500",
};

export function StatusBadge({ status }: { status: SandboxPhase }) {
  return (
    <Badge
      variant={status === "Failed" ? "destructive" : "outline"}
      className={cn("gap-1.5 font-medium", STYLES[status])}
    >
      <span className={cn("size-1.5 rounded-full", DOT[status])} />
      {status}
    </Badge>
  );
}
