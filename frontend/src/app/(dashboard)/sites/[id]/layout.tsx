import React from "react";
import { ArrowLeft } from "lucide-react";
import Link from "next/link";

interface SiteLayoutProps {
    children: React.ReactNode;
    params: Promise<{ id: string }>;
}

export default async function SiteLayout({
    children,
    params
}: Readonly<SiteLayoutProps>) {
    const { id } = await params;
    const siteId = parseInt(id);

    return (
        <div>
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3 pt-6 pl-6">
                    <Link
                        href="/sites"
                        className="text-muted-foreground hover:text-foreground flex items-center gap-2"
                    >
                        <ArrowLeft className="h-4 w-4 group-hover:-translate-x-0.5 transition-transform" />
                        All Sites
                    </Link>
                </div>
            </div>
            {children}
        </div>
    );
}