import { useEffect, useState } from "react";
import type { LogLine, Sandbox } from "@/models/sandbox";
import { ResourceExplorer } from "@/components/explorer/resource-explorer";
import { EventsTab } from "@/components/inspector/events-tab";
import { InspectorHeader } from "@/components/inspector/inspector-header";
import { LogsTab } from "@/components/inspector/logs-tab";
import { YamlTab } from "@/components/inspector/yaml-tab";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export function SandboxInspector({
  sandbox,
  logs,
  now,
  busy,
  onPause,
  onResume,
  onDelete,
}: {
  sandbox: Sandbox | null;
  logs: LogLine[];
  now: number;
  busy: boolean;
  onPause: () => void;
  onResume: () => void;
  onDelete: () => void;
}) {
  const [tab, setTab] = useState("logs");
  const [resourceId, setResourceId] = useState<string | null>(null);

  useEffect(() => {
    setResourceId(null);
  }, [sandbox?.name]);

  if (!sandbox) {
    return (
      <Empty className="h-full border-0">
        <EmptyHeader>
          <EmptyTitle>Select a sandbox</EmptyTitle>
          <EmptyDescription>
            Inspect live logs, a readable event timeline, and the underlying
            YAML. Owned Kubernetes resources appear in the explorer.
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <InspectorHeader
        sandbox={sandbox}
        busy={busy}
        onPause={onPause}
        onResume={onResume}
        onDelete={onDelete}
      />
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize="72%" minSize="40%">
          <Tabs
            value={tab}
            onValueChange={setTab}
            className="flex h-full min-h-0 gap-0"
          >
            <div className="flex h-8 shrink-0 items-center border-b px-2">
              <TabsList variant="line" className="h-8">
                <TabsTrigger value="logs">Logs</TabsTrigger>
                <TabsTrigger value="events">Events</TabsTrigger>
                <TabsTrigger value="yaml">YAML</TabsTrigger>
              </TabsList>
            </div>
            <TabsContent
              value="logs"
              className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
            >
              <LogsTab lines={logs} />
            </TabsContent>
            <TabsContent
              value="events"
              className="mt-0 min-h-0 flex-1 overflow-auto"
            >
              <EventsTab events={sandbox.events} now={now} />
            </TabsContent>
            <TabsContent
              value="yaml"
              className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
            >
              <YamlTab yaml={sandbox.yaml} />
            </TabsContent>
          </Tabs>
        </ResizablePanel>
        <ResizableHandle withHandle />
        <ResizablePanel defaultSize="28%" minSize="18%">
          <ResourceExplorer
            sandbox={sandbox}
            selectedId={resourceId}
            onSelect={setResourceId}
          />
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
