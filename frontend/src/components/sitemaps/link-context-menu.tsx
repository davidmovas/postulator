"use client";

import { useEffect, useRef } from "react";
import {
    ContextMenu,
    ContextMenuContent,
    ContextMenuItem,
    ContextMenuSeparator,
} from "@/components/ui/context-menu";
import { Check, X, Trash2 } from "lucide-react";
import { LinkStatus } from "@/models/linking";

interface LinkContextMenuProps {
    linkId: number;
    position: { x: number; y: number };
    status: LinkStatus;
    onApprove: (linkId: number) => Promise<boolean>;
    onReject: (linkId: number) => Promise<boolean>;
    onRemove: (linkId: number) => Promise<boolean>;
    onClose: () => void;
}

export function LinkContextMenu({
    linkId,
    position,
    status,
    onApprove,
    onReject,
    onRemove,
    onClose,
}: LinkContextMenuProps) {
    const menuRef = useRef<HTMLDivElement>(null);

    // Close menu when clicking outside
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
                onClose();
            }
        };

        const handleEscape = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                onClose();
            }
        };

        document.addEventListener("mousedown", handleClickOutside);
        document.addEventListener("keydown", handleEscape);

        return () => {
            document.removeEventListener("mousedown", handleClickOutside);
            document.removeEventListener("keydown", handleEscape);
        };
    }, [onClose]);

    const canApprove = status === "planned" || status === "rejected";
    const canReject = status === "planned" || status === "approved";

    return (
        <div
            ref={menuRef}
            className="fixed z-50 min-w-[160px] rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
            style={{
                left: position.x,
                top: position.y,
            }}
        >
            {canApprove && (
                <button
                    className="relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground"
                    onClick={() => onApprove(linkId)}
                >
                    <Check className="mr-2 h-4 w-4 text-green-500" />
                    <span>Approve Link</span>
                </button>
            )}
            {canReject && (
                <button
                    className="relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground"
                    onClick={() => onReject(linkId)}
                >
                    <X className="mr-2 h-4 w-4 text-yellow-500" />
                    <span>Reject Link</span>
                </button>
            )}
            {(canApprove || canReject) && (
                <div className="my-1 h-px bg-border" />
            )}
            <button
                className="relative flex w-full cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none hover:bg-accent hover:text-accent-foreground text-destructive"
                onClick={() => onRemove(linkId)}
            >
                <Trash2 className="mr-2 h-4 w-4" />
                <span>Delete Link</span>
            </button>
        </div>
    );
}
