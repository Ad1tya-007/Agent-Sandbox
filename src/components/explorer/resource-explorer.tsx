import { Box, FileText, HardDrive, Network, Workflow } from "lucide-react";
import type { ResourceKind, ResourceNode } from "@/models/resource";
import type { Sandbox } from "@/models/sandbox";
import { resourceDetailFor, resourceTreeFor } from "@/lib/resource-tree";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";

const ICONS: Record<ResourceKind, typeof Box> = {
  Sandbox: Box,
  Pod: Workflow,
  PersistentVolumeClaim: HardDrive,
  Service: Network,
  Events: FileText,
};

export function ResourceExplorer({
  sandbox,
  selectedId,
  onSelect,
}: {
  sandbox: Sandbox;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const tree = resourceTreeFor(sandbox);
  const selected = findNode(tree, selectedId ?? tree.id) ?? tree;
  const detail = resourceDetailFor(sandbox, selected);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex h-8 shrink-0 items-center border-b px-3">
        <h2 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
          Resources
        </h2>
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <ul className="p-2">
          <TreeItem
            node={tree}
            depth={0}
            selectedId={selected.id}
            onSelect={onSelect}
          />
        </ul>
      </ScrollArea>
      <Separator />
      <div className="shrink-0 p-3">
        <p className="mb-2 text-xs font-medium">
          {detail.kind} · {detail.name}
        </p>
        <dl className="space-y-1.5">
          {detail.fields.map((field) => (
            <div
              key={field.label}
              className="grid grid-cols-[5.5rem_1fr] gap-2"
            >
              <dt className="text-muted-foreground text-xs">{field.label}</dt>
              <dd className="truncate font-mono text-xs">{field.value}</dd>
            </div>
          ))}
        </dl>
      </div>
    </div>
  );
}

function TreeItem({
  node,
  depth,
  selectedId,
  onSelect,
}: {
  node: ResourceNode;
  depth: number;
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const Icon = ICONS[node.kind];
  const selected = node.id === selectedId;

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(node.id)}
        style={{ paddingLeft: 8 + depth * 14 }}
        className={cn(
          "flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-xs",
          selected
            ? "bg-muted text-foreground"
            : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
        )}
      >
        <Icon className="size-3.5 shrink-0" />
        <span className="truncate" title={node.name}>
          {node.kind}
        </span>
      </button>
      {node.children.length > 0 ? (
        <ul>
          {node.children.map((child) => (
            <TreeItem
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function findNode(node: ResourceNode, id: string): ResourceNode | null {
  if (node.id === id) return node;
  for (const child of node.children) {
    const found = findNode(child, id);
    if (found) return found;
  }
  return null;
}
