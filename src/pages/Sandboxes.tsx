import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { useConnection } from "@/hooks/use-connection";
import { useNow } from "@/hooks/use-now";
import { useSandboxCommands } from "@/hooks/use-sandbox-commands";
import { useSandboxLogs } from "@/hooks/use-sandbox-logs";
import { useSandboxes } from "@/hooks/use-sandboxes";
import { LiveBadge } from "@/components/connection-status";
import { SandboxInspector } from "@/components/inspector/sandbox-inspector";
import { CreateSandboxDialog } from "@/components/sandbox/create-sandbox-dialog";
import { DeleteSandboxDialog } from "@/components/sandbox/delete-sandbox-dialog";
import { SandboxTable } from "@/components/sandbox/sandbox-table";
import { Button } from "@/components/ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { PageHeader } from "@/layouts/page-header";

export function Sandboxes() {
  const connection = useConnection();
  const { sandboxes, hydrated } = useSandboxes();
  const { create, pause, resume, remove, pending } = useSandboxCommands();
  const now = useNow();
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteName, setDeleteName] = useState<string | null>(null);

  const selected =
    sandboxes.find((sandbox) => sandbox.name === selectedName) ?? null;
  const { lines } = useSandboxLogs(selected?.name ?? null);

  useEffect(() => {
    if (selectedName && !sandboxes.some((item) => item.name === selectedName)) {
      setSelectedName(null);
    }
  }, [sandboxes, selectedName]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader
        title="Sandboxes"
        meta={
          hydrated ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {sandboxes.length}
            </span>
          ) : null
        }
      >
        <LiveBadge state={connection.state} />
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus />
          Create
        </Button>
      </PageHeader>

      <ResizablePanelGroup orientation="vertical" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="42%" minSize="24%">
          <div className="h-full min-h-0 overflow-auto">
            <SandboxTable
              sandboxes={sandboxes}
              selectedName={selectedName}
              hydrated={hydrated}
              now={now}
              pending={pending}
              onSelect={setSelectedName}
              onPause={(name) => void pause(name)}
              onResume={(name) => void resume(name)}
              onDelete={setDeleteName}
              onCreate={() => setCreateOpen(true)}
            />
          </div>
        </ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel defaultSize="58%" minSize="28%">
          <SandboxInspector
            sandbox={selected}
            logs={lines}
            now={now}
            busy={selected ? Boolean(pending[selected.name]) : false}
            onPause={() => selected && void pause(selected.name)}
            onResume={() => selected && void resume(selected.name)}
            onDelete={() => selected && setDeleteName(selected.name)}
          />
        </ResizablePanel>
      </ResizablePanelGroup>

      <CreateSandboxDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        pending={Object.values(pending).includes("create")}
        onCreate={async (input) => {
          const result = await create(input);
          setSelectedName(result.name);
        }}
      />
      <DeleteSandboxDialog
        name={deleteName}
        open={deleteName != null}
        pending={deleteName ? pending[deleteName] === "delete" : false}
        onOpenChange={(open) => {
          if (!open) setDeleteName(null);
        }}
        onConfirm={() => {
          if (!deleteName) return;
          const name = deleteName;
          setDeleteName(null);
          void remove(name);
        }}
      />
    </div>
  );
}
