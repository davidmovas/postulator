"use client";
import { useEffect, useState } from "react";
import Image from "next/image";
import AppShell from "@/components/layout/AppShell";
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbList,
    BreadcrumbPage,
    BreadcrumbSeparator
} from "@/components/ui/breadcrumb";
import { RiScanLine } from "@remixicon/react";
import SitesPanel from "@/components/dashboard/SitesPanel";
import DashboardOverview from "@/components/dashboard/DashboardOverview";
import { NavigationProvider, useNavigation } from "@/context/navigation";
import { RiBardLine, RiCodeSSlashLine, RiSettings3Line, RiUserFollowLine } from "@remixicon/react";

function SplashScreen() {
    return (
        <div className="fixed inset-0 z-50 grid place-items-center bg-[var(--background)] text-[var(--foreground)]">
            <div className="flex flex-col items-center gap-4 animate-splash-in will-change-transform">
                <Image src="/appicon.svg" alt="App icon" width={120} height={120} className="drop-shadow-lg" />
                <h1 className="text-3xl sm:text-4xl font-semibold tracking-wide animate-title-reveal">Postulator</h1>
            </div>
        </div>
    );
}

function HeaderCrumbs() {
    const { section, setSection } = useNavigation();
    const title = section.charAt(0).toUpperCase() + section.slice(1);
    return (
        <Breadcrumb>
            <BreadcrumbList>
                <BreadcrumbItem className="hidden md:block">
                    <BreadcrumbLink href="#" onClick={(e)=>{e.preventDefault(); setSection("dashboard");}}>
                        <span className="sr-only">{title}</span>
                    </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator className="hidden md:block" />
                <BreadcrumbItem>
                    <BreadcrumbPage>{title}</BreadcrumbPage>
                </BreadcrumbItem>
            </BreadcrumbList>
        </Breadcrumb>
    );
}

function SectionContent() {
    const { section } = useNavigation();
    if (section === "dashboard") {
        return (
            <>
                <DashboardOverview />
            </>
        );
    }
    if (section === "sites") {
        return <SitesPanel />;
    }
    if (section === "jobs") {
        return (
            <div className="p-4 md:p-6 lg:p-8">
                <div className="text-sm text-muted-foreground">Schedule and monitor publishing jobs.</div>
                <h2 className="mt-1 text-2xl font-semibold tracking-tight">Jobs</h2>
                <p className="mt-2 text-muted-foreground">Cron builder and job list will appear here.</p>
            </div>
        );
    }
    if (section === "titles") {
        return (
            <div className="p-4 md:p-6 lg:p-8">
                <div className="text-sm text-muted-foreground">Browse and manage generated titles.</div>
                <h2 className="mt-1 text-2xl font-semibold tracking-tight">Titles</h2>
                <p className="mt-2 text-muted-foreground">Large, scalable list/grid will be placed here.</p>
            </div>
        );
    }
    // settings
    return (
        <div className="p-4 md:p-6 lg:p-8">
            <div className="text-sm text-muted-foreground">Application configuration.</div>
            <h2 className="mt-1 text-2xl font-semibold tracking-tight">Settings</h2>
            <p className="mt-2 text-muted-foreground">Settings forms will appear here.</p>
        </div>
    );
}

export default function Home() {
    const [showSplash, setShowSplash] = useState(false);

    useEffect(() => {
        // Play splash only once per app session
        const shown = typeof window !== "undefined" ? sessionStorage.getItem("splashShown") : "1";
        if (shown === "1") {
            setShowSplash(false);
            return;
        }
        setShowSplash(true);
        const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        const timeout = setTimeout(() => {
            setShowSplash(false);
            sessionStorage.setItem("splashShown", "1");
        }, prefersReduced ? 600 : 1700);
        return () => clearTimeout(timeout);
    }, []);

    return (
        <>
            {showSplash && <SplashScreen />}
            <NavigationProvider>
                <AppShell header={<HeaderCrumbs />}>
                    <SectionContent />
                </AppShell>
            </NavigationProvider>
        </>
    );
}
