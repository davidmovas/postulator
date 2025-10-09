"use client";
import AppShell from "@/components/layout/AppShell";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { RiBardLine } from "@remixicon/react";

export default function JobsPage() {
  return (
      <div className="p-4 md:p-6 lg:p-8">
          <h1 className="mt-1 text-2xl font-semibold tracking-tight">Jobs</h1>
          <p className="mt-2 text-muted-foreground">Cron builder and job list will appear here.</p>
      </div>
  );
}
