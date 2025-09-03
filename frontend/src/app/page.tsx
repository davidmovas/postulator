"use client";
import { useEffect, useState } from "react";
import Image from "next/image";

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
                <Image src="/appicon.svg" alt="App icon" width={100} height={100} />
                <span className="text-2xl sm:text-3xl font-medium">Postulator</span>
                <p className="text-sm text-[color:var(--foreground)]/80 max-w-prose">
                    Welcome! This is the home page of your Wails + Next application.
                </p>
            </main>
        </div>
    );
}
