"use client";

import { LightRays } from "@/components/ui/light-rays";

export function AppBackground() {
    return (
        <div className="fixed inset-0 -z-0 overflow-hidden">
            <LightRays
                count={5}
                color="rgba(255, 180, 100, 0.15)"
                blur={24}
                speed={20}
                length="100vh"
                className="w-full h-full"
            />
        </div>
    );
}