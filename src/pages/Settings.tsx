import { AppearanceSettings } from "@/components/settings/appearance-settings";
import { PageHeader } from "@/layouts/page-header";

export function Settings() {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Settings" />
      <div className="space-y-6 overflow-auto p-6">
        <p className="text-muted-foreground text-sm">
          Local appearance for this machine. Cluster connection comes with the
          Go backend.
        </p>
        <AppearanceSettings />
      </div>
    </div>
  );
}
