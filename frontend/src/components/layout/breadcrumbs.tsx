"use client";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbList,
    BreadcrumbPage,
    BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";

const titleMap: Record<string, string> = {
    dashboard: "Dashboard",
    jobs: "Jobs",
    sites: "Sites",
    articles: "Articles",
    categories: "Categories",
    topics: "Topics",
    prompts: "Prompts",
    providers: "Providers",
    settings: "Settings",
};

export function DynamicBreadcrumbs() {
    const pathname = usePathname();
    const segments = pathname.split("/").filter(Boolean);

    if (segments.length === 0) return null;

    return (
        <Breadcrumb>
            <BreadcrumbList>
                {segments.map((segment, index) => {
                    const isLast = index === segments.length - 1;
                    const href = "/" + segments.slice(0, index + 1).join("/");
                    const title = titleMap[segment] || segment;

                    return (
                        <div key={href} className="flex items-center gap-2">
                            {index > 0 && <BreadcrumbSeparator />}
                            <BreadcrumbItem>
                                {isLast ? (
                                    <BreadcrumbPage>{title}</BreadcrumbPage>
                                ) : (
                                    <BreadcrumbLink asChild>
                                        <Link href={href}>{title}</Link>
                                    </BreadcrumbLink>
                                )}
                            </BreadcrumbItem>
                        </div>
                    );
                })}
            </BreadcrumbList>
        </Breadcrumb>
    );
}