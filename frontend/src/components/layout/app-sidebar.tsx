"use client";
import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";

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
    RiLightbulbLine, RiArticleLine, RiChatThreadLine,
} from "@remixicon/react";

const APP_VERSION = "v1.0.0";

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
            { title: "Articles", href: "/articles", icon: RiArticleLine },
            { title: "Categories", href: "/categories", icon: RiChatThreadLine },
            { title: "Topics", href: "/topics", icon: RiLightbulbLine },
            { title: "Prompts", href: "/prompts", icon: RiChatAiLine },
            { title: "Providers", href: "/providers", icon: RiBardLine },
            { title: "Settings", href: "/settings", icon: RiSettings3Line },
        ],
    },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
    const pathname = usePathname();

    return (
        <Sidebar {...props}>
            <SidebarHeader>
                <div className="flex items-center text-center pt-2 pl-4">
                    <h1 className="text-3xl font-bold">Postulator</h1>
                </div>
            </SidebarHeader>
            <SidebarContent>
                {navItems.map((group) => (
                    <SidebarGroup key={group.title}>
                        <SidebarGroupLabel className="uppercase text-muted-foreground/60">
                            {group.title}
                        </SidebarGroupLabel>
                        <SidebarGroupContent className="px-2">
                            <SidebarMenu>
                                {group.items.map((item) => {
                                    const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
                                    return (
                                        <SidebarMenuItem key={item.title}>
                                            <SidebarMenuButton
                                                asChild
                                                className="group/menu-button font-medium gap-3 h-9 rounded-md bg-gradient-to-r hover:bg-transparent hover:from-sidebar-accent hover:to-sidebar-accent/40 data-[active=true]:from-primary/20 data-[active=true]:to-primary/5 [&>svg]:size-auto"
                                                isActive={isActive}
                                            >
                                                <Link href={item.href}>
                          <span className="inline-flex items-center gap-3">
                            {item.icon && (
                                <item.icon
                                    className="text-muted-foreground/60 group-data-[active=true]/menu-button:text-primary"
                                    size={22}
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
                    <span className="text-sm text-muted-foreground/60">{APP_VERSION}</span>
                </div>
            </SidebarFooter>

            <SidebarRail />
        </Sidebar>
    );
}