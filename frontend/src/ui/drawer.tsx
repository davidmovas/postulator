import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import { IconButton } from "./button.js";
import { CloseIcon } from "./icons/index.js";

export interface DrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    description?: string;
    closeLabel: string;
    width?: number;
    header?: ReactNode;
    footer?: ReactNode;
    children: ReactNode;
}

export function Drawer({
    open,
    onOpenChange,
    title,
    description,
    closeLabel,
    width = 520,
    header,
    footer,
    children,
}: DrawerProps): ReactElement {
    return (
        <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
            <RadixDialog.Portal>
                <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/45" />
                <RadixDialog.Content
                    style={{ width: `min(${width}px, 100vw)` }}
                    className={cx(
                        "fixed inset-y-0 right-0 z-50 flex flex-col border-l border-edge bg-panel",
                        "data-[state=open]:animate-slide-in-right",
                    )}
                >
                    <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
                        <RadixDialog.Title className="truncate text-xs font-semibold text-ink">
                            {title}
                        </RadixDialog.Title>
                        <div className="flex shrink-0 items-center gap-1">
                            {header}
                            <RadixDialog.Close asChild>
                                <IconButton icon={CloseIcon} label={closeLabel} variant="ghost" size="sm" />
                            </RadixDialog.Close>
                        </div>
                    </header>
                    {description === undefined ? (
                        <RadixDialog.Description className="sr-only">{title}</RadixDialog.Description>
                    ) : (
                        <RadixDialog.Description className="shrink-0 border-b border-hairline px-3 py-2 text-xs text-ink-dim">
                            {description}
                        </RadixDialog.Description>
                    )}
                    <div className="min-h-0 flex-1 overflow-auto">{children}</div>
                    {footer === undefined ? null : (
                        <footer className="flex shrink-0 justify-end gap-2 border-t border-hairline bg-inset px-3 py-2">
                            {footer}
                        </footer>
                    )}
                </RadixDialog.Content>
            </RadixDialog.Portal>
        </RadixDialog.Root>
    );
}
