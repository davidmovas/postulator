"use client";

import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";
import React from "react";
import { DynamicBreadcrumbs } from "@/components/layout/breadcrumbs";
import { AppSidebar } from "@/components/layout/app-sidebar";
import { AppBackground } from "@/components/layout/app-background";

export default function DashboardLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <SidebarProvider>
            <AppSidebar />
            <SidebarInset className="relative h-svh max-h-svh overflow-hidden">
                <AppBackground />
                <header className="flex h-16 shrink-0 items-center gap-2 border-b relative z-10">
                    <div className="flex flex-1 items-center gap-2 px-3">
                        <SidebarTrigger />
                        <Separator orientation="vertical" className="mr-2 h-4" />
                        <DynamicBreadcrumbs />
                    </div>
                </header>
                <div className="flex-1 min-h-0 overflow-auto relative z-10">
                    {children}
                </div>
            </SidebarInset>
        </SidebarProvider>
    );
}