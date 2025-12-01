"use client";
import * as React from "react";
import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { GetAppVersion } from "@/wailsjs/wailsjs/go/handlers/SettingsHandler";

import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarRail,
} from "@/components/ui/sidebar";
import {
    RiTimerLine,
    RiGlobalLine,
    RiBardLine,
    RiSettings3Line,
    RemixiconComponentType,
    RiChatAiLine,
    RiDashboard2Line,
    RiLightbulbLine, RiArticleLine,
} from "@remixicon/react";

const FALLBACK_VERSION = "dev";

type NavItem = {
    title: string;
    href: string;
    icon?: RemixiconComponentType;
};

type NavGroup = {
    title: string;
    items: NavItem[];
};

const navItems: NavGroup[] = [
    {
        title: "Sections",
        items: [
            { title: "Dashboard", href: "/dashboard", icon: RiDashboard2Line },
            { title: "Jobs", href: "/jobs", icon: RiTimerLine },
            { title: "Sites", href: "/sites", icon: RiGlobalLine },
           /* { title: "Articles", href: "/articles", icon: RiArticleLine },*/
           // { title: "Topics", href: "/topics", icon: RiLightbulbLine },
            { title: "Prompts", href: "/prompts", icon: RiChatAiLine },
            { title: "AI Providers", href: "/ai-providers", icon: RiBardLine },
            { title: "Settings", href: "/settings", icon: RiSettings3Line },
        ],
    },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
    const pathname = usePathname();
    const [version, setVersion] = useState(FALLBACK_VERSION);

    useEffect(() => {
        GetAppVersion()
            .then((response) => {
                if (response.success && response.data) {
                    const v = response.data.version;
                    setVersion(v.startsWith("v") ? v : `v${v}`);
                }
            })
            .catch(() => {
                // Keep fallback version on error
            });
    }, []);

    return (
        <Sidebar
            {...props}
            className="bg-zinc-900 border-r border-zinc-800 shadow-md"
        >
            <SidebarHeader>
                <div className="flex items-center text-center pt-2 pl-4">
                    <h1 className="text-3xl font-bold text-white">Postulator</h1>
                </div>
            </SidebarHeader>

            <SidebarContent>
                {navItems.map((group) => (
                    <SidebarGroup key={group.title}>
                        <SidebarGroupLabel className="uppercase text-zinc-500 tracking-wide">
                            {group.title}
                        </SidebarGroupLabel>

                        <SidebarGroupContent className="px-2">
                            <SidebarMenu>
                                {group.items.map((item) => {
                                    const isActive =
                                        pathname === item.href ||
                                        pathname.startsWith(item.href + "/");

                                    return (
                                        <SidebarMenuItem key={item.title}>
                                            <SidebarMenuButton
                                                asChild
                                                isActive={isActive}
                                                className={`
                                                    group/menu-button flex items-center gap-3 h-9 rounded-md
                                                    font-medium transition-colors duration-200
                                                    ${
                                                    isActive
                                                        ? "bg-transparent"
                                                        : "text-zinc-300 hover:bg-zinc-800 hover:text-white"
                                                }
                                                `}
                                            >
                                                <Link href={item.href}>
                                                    <span className="inline-flex items-center gap-3">
                                                        {item.icon && (
                                                            <item.icon
                                                                className={`
                                                                    size-5
                                                                    ${
                                                                    isActive
                                                                        ? "text-primary"
                                                                        : "text-zinc-400 group-hover/menu-button:text-white"
                                                                }
                                                                `}
                                                                aria-hidden="true"
                                                            />
                                                        )}
                                                        <span>{item.title}</span>
                                                    </span>
                                                </Link>
                                            </SidebarMenuButton>
                                        </SidebarMenuItem>
                                    );
                                })}
                            </SidebarMenu>
                        </SidebarGroupContent>
                    </SidebarGroup>
                ))}
            </SidebarContent>

            <SidebarFooter>
                <div className="flex items-center justify-start p-4">
                    <span className="text-sm text-zinc-500">{version}</span>
                </div>
            </SidebarFooter>

            <SidebarRail />
        </Sidebar>
    );
}
