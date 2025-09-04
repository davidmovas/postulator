"use client";
import AppShell from "@/components/layout/AppShell";
import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, BreadcrumbList, BreadcrumbPage, BreadcrumbSeparator } from "@/components/ui/breadcrumb";
import { RiUserFollowLine } from "@remixicon/react";
import SitesPanel from "@/components/dashboard/SitesPanel";

export default function SitesPage() {
  return (
    <AppShell
      header={(
        <Breadcrumb>
          <BreadcrumbList>
            <BreadcrumbItem className="hidden md:block">
              <BreadcrumbLink href="/">
                <RiUserFollowLine size={22} aria-hidden="true" />
                <span className="sr-only">Sites</span>
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator className="hidden md:block" />
            <BreadcrumbItem>
              <BreadcrumbPage>Sites</BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>
      )}
    >
      <SitesPanel />
    </AppShell>
  );
}
