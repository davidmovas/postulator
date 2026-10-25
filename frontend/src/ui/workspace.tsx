import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactElement, ReactNode } from "react";

import { IconButton } from "./button.js";
import { CloseIcon } from "./icons/index.js";

export interface WorkspaceProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    description?: string;
    closeLabel: string;
    footer?: ReactNode;
    children: ReactNode;
}

export function Workspace({
    open,
    onOpenChange,
    title,
    description,
    closeLabel,
    footer,
    children,
}: WorkspaceProps): ReactElement {
    return (
        <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
            <RadixDialog.Portal>
                <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/55" />
                <RadixDialog.Content
                    data-workspace={true}
                    className="fixed inset-3 z-50 mx-auto flex max-w-[1440px] flex-col overflow-hidden rounded-lg border border-edge bg-panel data-[state=open]:animate-fade-in"
                >
                    <header className="flex h-10 shrink-0 items-center gap-3 border-b border-hairline pr-2 pl-4">
                        <RadixDialog.Title className="min-w-0 flex-1 truncate text-sm font-semibold text-ink">
                            {title}
                        </RadixDialog.Title>
                        <RadixDialog.Close asChild>
                            <IconButton icon={CloseIcon} label={closeLabel} variant="ghost" size="sm" />
                        </RadixDialog.Close>
                    </header>
                    {description === undefined ? (
                        <RadixDialog.Description className="sr-only">{title}</RadixDialog.Description>
                    ) : (
                        <RadixDialog.Description className="shrink-0 border-b border-hairline px-4 py-2 text-xs text-ink-dim">
                            {description}
                        </RadixDialog.Description>
                    )}
                    <div className="flex min-h-0 flex-1 flex-col">{children}</div>
                    {footer === undefined ? null : (
                        <footer className="flex shrink-0 items-center gap-2 border-t border-hairline bg-inset px-4 py-2">
                            {footer}
                        </footer>
                    )}
                </RadixDialog.Content>
            </RadixDialog.Portal>
        </RadixDialog.Root>
    );
}
