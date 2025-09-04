"use client";
import AppShell from "@/components/layout/AppShell";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { RiSettings3Line } from "@remixicon/react";

export default function SettingsPage() {
  return (
    <AppShell
      header={(
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/">
                <RiSettings3Line size={22} aria-hidden="true" />
                <span className="sr-only">Settings</span>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>Settings</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      )}
    >
      <div className="p-4 md:p-6 lg:p-8">
        <div className="text-sm text-muted-foreground">Application configuration.</div>
        <h2 className="mt-1 text-2xl font-semibold tracking-tight">Settings</h2>
        <p className="mt-2 text-muted-foreground">Settings forms will appear here.</p>
      </div>
    </AppShell>
  );
}
