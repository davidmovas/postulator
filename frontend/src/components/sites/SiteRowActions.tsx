"use client";

import React from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Activity, Copy as CopyIcon, ExternalLink, Lock, MoreVertical, Pencil, Trash2 } from "lucide-react";
import { Site } from "@/services/site";

export interface SiteRowActionsProps {
  site: Site;
  disabled?: boolean;
  onOpenDefault: (url: string) => void;
  onOpenTor: (url: string) => void;
  onCopyUrl: (url: string) => void | Promise<void>;
  onCheckHealth: (siteId: number) => void | Promise<void>;
  onEdit: (site: Site) => void;
  onRequestPassword: (site: Site) => void;
  onRequestDelete: (siteId: number) => void;
}

export function SiteRowActions({
  site,
  disabled = false,
  onOpenDefault,
  onOpenTor,
  onCopyUrl,
  onCheckHealth,
  onEdit,
  onRequestPassword,
  onRequestDelete,
}: SiteRowActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={disabled}>
          <MoreVertical className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => onOpenDefault(site.url)}>
          <ExternalLink className="h-4 w-4 mr-2" />
          Open (Default Browser)
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onOpenTor(site.url)}>
          <ExternalLink className="h-4 w-4 mr-2" />
          Open in Tor
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onCopyUrl(site.url)}>
          <CopyIcon className="h-4 w-4 mr-2" />
          Copy URL
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onCheckHealth(site.id)}>
          <Activity className="h-4 w-4 mr-2" />
          Check Health
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onRequestPassword(site)}>
          <Lock className="h-4 w-4 mr-2" />
          Set Password
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => onEdit(site)}>
          <Pencil className="h-4 w-4 mr-2" />
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => onRequestDelete(site.id)}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 className="h-4 w-4 mr-2" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default SiteRowActions;
