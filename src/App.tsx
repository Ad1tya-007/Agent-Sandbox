import { useState, type ComponentType } from "react";
import { Box, Settings as SettingsIcon } from "lucide-react";
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
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/components/settings/theme-provider";
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

export default function App() {
  const [page, setPage] = useState<PageKey>("sandboxes");

  return (
    <TooltipProvider delayDuration={0}>
      <ThemeProvider>
        <SidebarProvider defaultOpen className="relative">
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
              <p className="text-sidebar-foreground/70 px-2 py-1.5 text-xs group-data-[collapsible=icon]:hidden">
                No cluster connected
              </p>
            </SidebarFooter>
            <SidebarRail />
          </Sidebar>
          <SidebarInset>
            <div className="flex flex-1 flex-col gap-4 p-6">
              <PageView page={page} />
            </div>
          </SidebarInset>
          <SidebarTrigger className="absolute top-1 right-4 z-10" />
        </SidebarProvider>
        <Toaster closeButton position="top-right" />
      </ThemeProvider>
    </TooltipProvider>
  );
}
