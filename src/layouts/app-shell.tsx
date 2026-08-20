import { useState, type ComponentType } from "react";
import { Box, Settings as SettingsIcon } from "lucide-react";
import { ConnectionStatus } from "@/components/connection-status";
import { ThemeProvider } from "@/components/settings/theme-provider";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
} from "@/components/ui/sidebar";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useConnection } from "@/hooks/use-connection";
import { Sandboxes } from "@/pages/Sandboxes";
import { Settings } from "@/pages/Settings";

type PageKey = "sandboxes" | "settings";

const NAV_ITEMS: {
  key: PageKey;
  label: string;
  icon: ComponentType<{ className?: string }>;
}[] = [
  { key: "sandboxes", label: "Sandboxes", icon: Box },
  { key: "settings", label: "Settings", icon: SettingsIcon },
];

function PageView({ page }: { page: PageKey }) {
  switch (page) {
    case "settings":
      return <Settings />;
    default:
      return <Sandboxes />;
  }
}

function Shell() {
  const [page, setPage] = useState<PageKey>("sandboxes");
  const connection = useConnection();

  return (
    <SidebarProvider defaultOpen className="h-svh overflow-hidden">
      <Sidebar collapsible="icon">
        <SidebarHeader className="border-sidebar-border flex flex-row items-center border-b">
          <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 shrink-0 items-center justify-center rounded-lg text-xs font-semibold">
            AS
          </div>
          <div className="grid min-w-0 flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
            <span className="truncate font-semibold">Agent Sandbox</span>
            <span className="text-sidebar-foreground/70 truncate text-xs">
              Desktop
            </span>
          </div>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Workloads</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {NAV_ITEMS.map(({ key, label, icon: Icon }) => (
                  <SidebarMenuItem key={key}>
                    <SidebarMenuButton
                      isActive={page === key}
                      tooltip={label}
                      onClick={() => setPage(key)}
                    >
                      <Icon />
                      <span>{label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter className="border-sidebar-border border-t">
          <ConnectionStatus connection={connection} />
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset className="min-h-0 overflow-hidden">
        <PageView page={page} />
      </SidebarInset>
    </SidebarProvider>
  );
}

export default function AppShell() {
  return (
    <TooltipProvider delayDuration={0}>
      <ThemeProvider>
        <Shell />
        <Toaster closeButton position="top-right" />
      </ThemeProvider>
    </TooltipProvider>
  );
}
