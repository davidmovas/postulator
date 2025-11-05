"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import {
    Breadcrumb,
    BreadcrumbList,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbPage,
    BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import React, { useState, useEffect } from "react";
import { siteService } from "@/services/sites";

export function DynamicBreadcrumbs() {
    const pathname = usePathname();
    const segments = pathname.split("/").filter(Boolean);
    const [siteNames, setSiteNames] = useState<Record<string, string>>({});

    useEffect(() => {
        const loadSiteNames = async () => {
            const newSiteNames: Record<string, string> = {};

            segments.forEach((segment, index) => {
                if (index > 0 && segments[index - 1] === "sites" && /^\d+$/.test(segment)) {
                    const siteId = segment;
                    if (!siteNames[siteId]) {
                        siteService.getSite(parseInt(siteId))
                            .then(site => {
                                setSiteNames(prev => ({
                                    ...prev,
                                    [siteId]: site.name
                                }));
                            })
                            .catch(() => {
                                setSiteNames(prev => ({
                                    ...prev,
                                    [siteId]: `Site ${siteId}`
                                }));
                            });
                    }
                }
            });
        };

        loadSiteNames();
    }, [pathname, segments]);

    if (segments.length === 0) return null;

    const getTitle = (segment: string, index: number): string => {
        if (index > 0 && segments[index - 1] === "sites" && /^\d+$/.test(segment)) {
            return siteNames[segment] || `Site ${segment}`;
        }

        const titleMap: Record<string, string> = {
            "": "Home",
            "dashboard": "Dashboard",
            "sites": "Sites",
            "articles": "Articles",
            "jobs": "Jobs",
            "topics": "Topics",
            "categories": "Categories",
            "prompts": "Prompts",
            "providers": "Providers",
            "settings": "Settings"
        };

        return titleMap[segment] || segment.charAt(0).toUpperCase() + segment.slice(1);
    };

    return (
        <Breadcrumb>
            <BreadcrumbList>
                {segments.map((segment, index) => {
                    const isLast = index === segments.length - 1;
                    const href = "/" + segments.slice(0, index + 1).join("/");
                    const title = getTitle(segment, index);

                    return (
                        <React.Fragment key={href}>
                            <BreadcrumbItem>
                                {isLast ? (
                                    <BreadcrumbPage>{title}</BreadcrumbPage>
                                ) : (
                                    <BreadcrumbLink asChild>
                                        <Link href={href}>{title}</Link>
                                    </BreadcrumbLink>
                                )}
                            </BreadcrumbItem>
                            {!isLast && <BreadcrumbSeparator />}
                        </React.Fragment>
                    );
                })}
            </BreadcrumbList>
        </Breadcrumb>
    );
}