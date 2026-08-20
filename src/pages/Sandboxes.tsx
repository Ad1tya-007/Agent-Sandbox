export function Sandboxes() {
  return (
    <div className="flex flex-1 flex-col items-start justify-center gap-2 py-16">
      <h1 className="text-2xl font-semibold tracking-tight">Sandboxes</h1>
      <p className="text-muted-foreground max-w-md text-sm leading-relaxed">
        Connect to a Kubernetes cluster to list, create, pause, resume, and
        delete Agent Sandboxes in real time. Cluster access will go through the
        Go backend — this UI never talks to Kubernetes directly.
      </p>
    </div>
  );
}
