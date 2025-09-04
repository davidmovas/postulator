"use client";
import AppShell from "@/components/layout/AppShell";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { RiCodeSSlashLine } from "@remixicon/react";

export default function TitlesPage() {
  return (
    <AppShell
      header={(
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/">
                <RiCodeSSlashLine size={22} aria-hidden="true" />
                <span className="sr-only">Titles</span>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>Titles</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      )}
    >
      <div className="p-4 md:p-6 lg:p-8">
        <div className="text-sm text-muted-foreground">Browse and manage generated titles.</div>
        <h2 className="mt-1 text-2xl font-semibold tracking-tight">Titles</h2>
        <p className="mt-2 text-muted-foreground">Large, scalable list/grid will be placed here.</p>
      </div>
    </AppShell>
  );
}
