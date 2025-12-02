"use client";

import { useEffect } from "react";
import { Unlink } from "lucide-react";

interface MenuPosition {
    x: number;
    y: number;
}

interface EdgeContextMenuProps {
    position: MenuPosition | null;
    onClose: () => void;
    onDeleteEdge: () => void;
}

export function EdgeContextMenu({
    position,
    onClose,
    onDeleteEdge,
}: EdgeContextMenuProps) {
    // Close on click outside or escape
    useEffect(() => {
        const handleClick = () => onClose();
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === "Escape") onClose();
        };

        if (position) {
            document.addEventListener("click", handleClick);
            document.addEventListener("keydown", handleKeyDown);
        }

        return () => {
            document.removeEventListener("click", handleClick);
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [position, onClose]);

    if (!position) return null;

    return (
        <div
            className="fixed z-50 min-w-[160px] bg-popover border rounded-md shadow-md py-1 animate-in fade-in-0 zoom-in-95"
            style={{
                left: position.x,
                top: position.y,
            }}
            onClick={(e) => e.stopPropagation()}
        >
            <button
                className="w-full flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-accent transition-colors text-left text-destructive hover:text-destructive"
                onClick={() => {
                    onDeleteEdge();
                    onClose();
                }}
            >
                <Unlink className="h-4 w-4" />
                Remove Connection
            </button>
        </div>
    );
}
