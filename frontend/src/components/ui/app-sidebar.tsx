import * as React from "react";

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
    RiScanLine,
    RiTimerLine,
    RiGlobalLine,
    RiArticleLine,
    RiBardLine,
    RiSettings3Line,
    RemixiconComponentType,
} from "@remixicon/react";

import { useNavigation, type Section } from "@/context/navigation";

type LeafItem = {
    title: string;
    key: Section;
    icon?: RemixiconComponentType;
};

type NavGroup = {
    title: string;
    items: LeafItem[];
};

const navItems: NavGroup[] = [
    {
        title: "Sections",
        items: [
            {
                title: "Dashboard",
                key: "dashboard",
                icon: RiScanLine,
            },
            {
                title: "Jobs",
                key: "jobs",
                icon: RiTimerLine,
            },
            {
                title: "Sites",
                key: "sites",
                icon: RiGlobalLine,
            },
            {
                title: "Topics",
                key: "topics",
                icon: RiArticleLine,
            },
            {
                title: "Prompts",
                key: "prompts",
                icon: RiBardLine,
            },
            {
                title: "Settings",
                key: "settings",
                icon: RiSettings3Line,
            },
        ],
    },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { section, setSection } = useNavigation();
  return (
    <Sidebar {...props}>
      <SidebarHeader>
          <div className="flex items-center text-center pt-2 pl-4">
              <h1 className="text-3xl font-bold">Postulator</h1>
          </div>
      </SidebarHeader>
      <SidebarContent>
        {/* We create a SidebarGroup for each parent. */}
        {navItems.map((group) => (
          <SidebarGroup key={group.title}>
            <SidebarGroupLabel className="uppercase text-muted-foreground/60">
              {group.title}
            </SidebarGroupLabel>
            <SidebarGroupContent className="px-2">
              <SidebarMenu>
                {group.items && group.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      className="group/menu-button font-medium gap-3 h-9 rounded-md bg-gradient-to-r hover:bg-transparent hover:from-sidebar-accent hover:to-sidebar-accent/40 data-[active=true]:from-primary/20 data-[active=true]:to-primary/5 [&>svg]:size-auto"
                      isActive={section === item.key}
                      onClick={(e) => {
                        e.preventDefault();
                        setSection(item.key);
                      }}
                    >
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
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}
