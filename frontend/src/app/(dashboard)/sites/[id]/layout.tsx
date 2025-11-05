import React from "react";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

export default function SiteLayout({
    children,
}: Readonly<{
    children: React.ReactNode;
}>) {
    return (
        <div>
            <div className="flex items-center justify-between">
                <div className="pt-6 pl-6">
                    <Link
                        href="/sites"
                        className="text-muted-foreground hover:text-foreground flex items-center gap-2"
                    >
                        <ArrowLeft className="h-4 w-4" /> Back to site list
                    </Link>
                </div>
            </div>
            {children}
        </div>
    );
}