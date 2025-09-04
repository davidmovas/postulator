"use client";
import { useEffect, useState } from "react";
import Image from "next/image";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/ui/app-sidebar";

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

export default function Home() {
    const [showSplash, setShowSplash] = useState(true);

    useEffect(() => {
        // Keep splash for a short moment; respect reduced motion by shortening
        const prefersReduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
        const timeout = setTimeout(() => setShowSplash(false), prefersReduced ? 600 : 1700);
        return () => clearTimeout(timeout);
    }, []);

    return (
        <div className="relative min-h-screen grid grid-rows-[20px_1fr_20px] items-center justify-items-center p-8 pb-20 gap-16 sm:p-20 font-[family-name:var(--font-geist-sans)]">
            {showSplash && <SplashScreen />}
            <main className={`flex flex-col gap-8 row-start-2 items-center sm:items-start transition-opacity duration-500 ${showSplash ? "opacity-0" : "opacity-100"}`}>
                <SidebarProvider>
                    <AppSidebar />
                    <SidebarInset className="overflow-hidden px-4 md:px-6 lg:px-8">
                        <p>CONTENT HERE</p>
                    </SidebarInset>
                </SidebarProvider>
            </main>
        </div>
    );
}
