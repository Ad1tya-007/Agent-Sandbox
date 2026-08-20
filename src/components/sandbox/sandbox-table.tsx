import { MoreHorizontal, Pause, Play, Trash2 } from "lucide-react";
import type { Sandbox } from "@/models/sandbox";
import { formatAge } from "@/lib/format";
import { cn } from "@/lib/utils";
import { StatusBadge } from "@/components/sandbox/status-badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function SandboxTable({
  sandboxes,
  selectedName,
  hydrated,
  now,
  pending,
  onSelect,
  onPause,
  onResume,
  onDelete,
  onCreate,
}: {
  sandboxes: Sandbox[];
  selectedName: string | null;
  hydrated: boolean;
  now: number;
  pending: Record<string, string>;
  onSelect: (name: string) => void;
  onPause: (name: string) => void;
  onResume: (name: string) => void;
  onDelete: (name: string) => void;
  onCreate: () => void;
}) {
  if (!hydrated) {
    return (
      <div className="space-y-2 p-3">
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-full" />
        ))}
      </div>
    );
  }

  if (sandboxes.length === 0) {
    return (
      <Empty className="h-full border-0">
        <EmptyHeader>
          <EmptyTitle>No sandboxes</EmptyTitle>
          <EmptyDescription>
            Create a sandbox to run a workload on the connected cluster.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button size="sm" onClick={onCreate}>
            Create sandbox
          </Button>
        </EmptyContent>
      </Empty>
    );
  }

  return (
    <Table>
      <TableHeader className="bg-background sticky top-0 z-10">
        <TableRow className="hover:bg-transparent">
          <TableHead>Name</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Image</TableHead>
          <TableHead>CPU</TableHead>
          <TableHead>Memory</TableHead>
          <TableHead>Age</TableHead>
          <TableHead>Node</TableHead>
          <TableHead className="w-10" />
        </TableRow>
      </TableHeader>
      <TableBody>
        {sandboxes.map((sandbox) => {
          const selected = sandbox.name === selectedName;
          const busy = Boolean(pending[sandbox.name]);
          return (
            <TableRow
              key={sandbox.name}
              data-state={selected ? "selected" : undefined}
              className={cn(
                "cursor-pointer",
                sandbox.status === "Terminating" && "opacity-60",
              )}
              onClick={() => onSelect(sandbox.name)}
            >
              <TableCell className="font-medium">{sandbox.name}</TableCell>
              <TableCell>
                <StatusBadge status={sandbox.status} />
              </TableCell>
              <TableCell className="text-muted-foreground max-w-[14rem] truncate font-mono text-xs">
                {sandbox.image}
              </TableCell>
              <TableCell className="tabular-nums">{sandbox.cpu}</TableCell>
              <TableCell className="tabular-nums">{sandbox.memory}</TableCell>
              <TableCell className="text-muted-foreground tabular-nums">
                {formatAge(sandbox.createdAt, now)}
              </TableCell>
              <TableCell className="text-muted-foreground">
                {sandbox.node ?? "—"}
              </TableCell>
              <TableCell className="text-right">
                <DropdownMenu>
                  <DropdownMenuTrigger
                    asChild
                    onClick={(event) => event.stopPropagation()}
                  >
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      aria-label={`${sandbox.name} actions`}
                      disabled={busy || sandbox.status === "Terminating"}
                    >
                      <MoreHorizontal />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent
                    align="end"
                    onClick={(event) => event.stopPropagation()}
                  >
                    {sandbox.status === "Paused" ? (
                      <DropdownMenuItem
                        disabled={busy}
                        onClick={() => onResume(sandbox.name)}
                      >
                        <Play />
                        Resume
                      </DropdownMenuItem>
                    ) : (
                      <DropdownMenuItem
                        disabled={busy || sandbox.status !== "Running"}
                        onClick={() => onPause(sandbox.name)}
                      >
                        <Pause />
                        Pause
                      </DropdownMenuItem>
                    )}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      variant="destructive"
                      disabled={busy}
                      onClick={() => onDelete(sandbox.name)}
                    >
                      <Trash2 />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
