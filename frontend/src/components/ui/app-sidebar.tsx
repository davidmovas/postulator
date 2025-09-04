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
    RiBardLine,
    RiUserFollowLine,
    RiCodeSSlashLine,
    RiSettings3Line,
    RiLogoutBoxLine, RemixiconComponentType,
} from "@remixicon/react";

interface NavItem {
    title: string;
    url: string;
    icon?: RemixiconComponentType;
    isActive?: boolean;
    items?: NavItem[];
}

const navItems: NavItem[] = [
    {
        title: "Sections",
        url: "#",
        items: [
            {
                title: "Dashboard",
                url: "/",
                icon: RiScanLine,
            },
            {
                title: "Jobs",
                url: "#",
                icon: RiBardLine,
            },
            {
                title: "Sites",
                url: "#",
                icon: RiUserFollowLine,
            },
            {
                title: "Titles",
                url: "#",
                icon: RiCodeSSlashLine,
            },
            {
                title: "Settings",
                url: "#",
                icon: RiSettings3Line,
            },
        ],
    },
];

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  return (
    <Sidebar {...props}>
      <SidebarHeader>
          <div className="flex items-center text-center pt-2 pl-4">
              <h1 className="text-3xl font-bold">Postulator</h1>
          </div>
      </SidebarHeader>
      <SidebarContent>
        {/* We create a SidebarGroup for each parent. */}
        {navItems.map((item) => (
          <SidebarGroup key={item.title}>
            <SidebarGroupLabel className="uppercase text-muted-foreground/60">
              {item.title}
            </SidebarGroupLabel>
            <SidebarGroupContent className="px-2">
              <SidebarMenu>
                {item.items && item.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton
                      asChild
                      className="group/menu-button font-medium gap-3 h-9 rounded-md bg-gradient-to-r hover:bg-transparent hover:from-sidebar-accent hover:to-sidebar-accent/40 data-[active=true]:from-primary/20 data-[active=true]:to-primary/5 [&>svg]:size-auto"
                      isActive={item.isActive}
                    >
                      <a href={item.url}>
                        {item.icon && (
                          <item.icon
                            className="text-muted-foreground/60 group-data-[active=true]/menu-button:text-primary"
                            size={22}
                            aria-hidden="true"
                          />
                        )}
                        <span>{item.title}</span>
                      </a>
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
