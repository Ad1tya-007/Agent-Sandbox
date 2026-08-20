import { AppearanceSettings } from "@/components/settings/appearance-settings";

export function Settings() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-muted-foreground text-sm">
          Local appearance for this machine. Cluster connection comes with the
          Go backend.
        </p>
      </div>
      <AppearanceSettings />
    </div>
  );
}
