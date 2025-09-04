"use client";
import AppShell from "@/components/layout/AppShell";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { RiBardLine } from "@remixicon/react";

export default function JobsPage() {
  return (
    <AppShell
      header={(
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/">
                <RiBardLine size={22} aria-hidden="true" />
                <span className="sr-only">Jobs</span>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>Jobs</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      )}
    >
      <div className="p-4 md:p-6 lg:p-8">
        <div className="text-sm text-muted-foreground">Schedule and monitor publishing jobs.</div>
        <h2 className="mt-1 text-2xl font-semibold tracking-tight">Jobs</h2>
        <p className="mt-2 text-muted-foreground">Cron builder and job list will appear here.</p>
      </div>
    </AppShell>
  );
}
