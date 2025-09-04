"use client";
import * as React from "react";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/ui/app-sidebar";
import { Separator } from "@/components/ui/separator";
import { NavigationProvider } from "@/context/navigation";

export type Crumb = {
  label: string;
  href?: string;
  icon?: React.ReactNode;
  current?: boolean;
};

export interface AppShellProps {
  header?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

/**
 * AppShell provides a reusable layout with sidebar and top header area.
 * Pass a custom header (e.g., breadcrumbs) via the header prop.
 */
export function AppShell({ header, children, className }: AppShellProps) {
  return (
    <NavigationProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className={className}>
          <header className="flex h-16 shrink-0 items-center gap-2 border-b">
            <div className="flex flex-1 items-center gap-2 px-3">
              <SidebarTrigger />
              <Separator orientation="vertical" className="mr-2 data-[orientation=vertical]:h-4" />
              {header}
            </div>
          </header>
          <div className="flex-1 min-h-0">
            {children}
          </div>
        </SidebarInset>
      </SidebarProvider>
    </NavigationProvider>
  );
}

export default AppShell;
