import type { ReactNode } from "react";
import { SidebarTrigger } from "@/components/ui/sidebar";

export function PageHeader({
  title,
  meta,
  children,
}: {
  title: string;
  meta?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <header className="flex h-11 shrink-0 items-center gap-2 border-b px-3">
      <SidebarTrigger className="-ml-1" />
      <div className="flex min-w-0 items-baseline gap-2">
        <h1 className="text-sm font-semibold tracking-tight">{title}</h1>
        {meta}
      </div>
      <div className="ml-auto flex items-center gap-2">{children}</div>
    </header>
  );
}
