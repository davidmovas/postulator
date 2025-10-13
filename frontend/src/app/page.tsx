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
import { NavigationProvider, useNavigation } from "@/context/navigation";
import JobsPage from "@/app/jobs/page";
import SitesPage from "@/app/sites/page";
import DashboardPage from "@/app/dashboard/page";
import SettingsPage from "@/app/settings/page";
import TopicsPage from "@/app/topics/page";
import PromptsPage from "@/app/prompts/page";
import AIProvidersPage from "@/app/ai-providers/page";

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
    if (section === "sites") {
        return <SitesPage />;
    }
    if (section === "jobs") {
        return <JobsPage />;
    }
    if (section === "topics") {
        return <TopicsPage />;
    }
    if (section === "prompts") {
        return <PromptsPage />;
    }
    if (section === "ai-providers") {
        return <AIProvidersPage />;
    }
    if (section === "settings") {
        return <SettingsPage />;
    }
    return <DashboardPage />;
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
