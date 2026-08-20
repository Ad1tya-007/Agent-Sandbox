import type { Connection } from "@/models/connection";
import { cn } from "@/lib/utils";
import { Spinner } from "@/components/ui/spinner";

export function ConnectionStatus({ connection }: { connection: Connection }) {
  const color =
    connection.state === "connected"
      ? "bg-emerald-500"
      : connection.state === "error"
        ? "bg-destructive"
        : connection.state === "connecting"
          ? "bg-amber-500"
          : "bg-muted-foreground";

  const label =
    connection.state === "connected"
      ? (connection.cluster ?? "Connected")
      : connection.state === "connecting"
        ? "Connecting"
        : connection.state === "error"
          ? (connection.message ?? "Error")
          : "Disconnected";

  return (
    <div
      className="text-sidebar-foreground/80 flex items-center gap-2 px-2 py-1.5 text-xs"
      title={connection.message ?? undefined}
    >
      {connection.state === "connecting" ? (
        <Spinner className="size-3" />
      ) : (
        <span className={cn("size-1.5 shrink-0 rounded-full", color)} />
      )}
      <span className="truncate group-data-[collapsible=icon]:hidden">
        {label}
      </span>
    </div>
  );
}

export function LiveBadge({ state }: { state: Connection["state"] }) {
  if (state === "connecting") {
    return (
      <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
        <Spinner className="size-3" />
        Connecting
      </span>
    );
  }

  if (state !== "connected") {
    return (
      <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
        <span className="bg-muted-foreground size-1.5 rounded-full" />
        Offline
      </span>
    );
  }

  return (
    <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
      <span className="size-1.5 rounded-full bg-emerald-500" />
      Live
    </span>
  );
}
