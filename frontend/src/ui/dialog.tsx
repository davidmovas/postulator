import * as RadixDialog from "@radix-ui/react-dialog";
import type { ReactElement, ReactNode } from "react";

import { Button } from "./button.js";
import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";
import { toneClasses } from "./tone.js";

export interface DialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    title: string;
    description: string;
    confirmLabel: string;
    onConfirm: () => void;
    cancelLabel: string;
    destructive?: boolean;
    icon?: IconComponent;
    busy?: boolean;
    children?: ReactNode;
}

export function Dialog({
    open,
    onOpenChange,
    title,
    description,
    confirmLabel,
    onConfirm,
    cancelLabel,
    destructive = false,
    icon: Icon,
    busy = false,
    children,
}: DialogProps): ReactElement {
    return (
        <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
            <RadixDialog.Portal>
                <RadixDialog.Overlay className="fixed inset-0 z-40 bg-black/55 data-[state=open]:animate-fade-in" />
                <RadixDialog.Content className="data-[state=open]:animate-fade-in fixed top-1/2 left-1/2 z-50 w-[min(28rem,calc(100vw-3rem))] -translate-x-1/2 -translate-y-1/2 overflow-hidden rounded-lg border border-edge bg-inset">
                    <div className="flex flex-col gap-2 px-4 py-3.5">
                        <RadixDialog.Title className="flex items-center gap-2 text-base font-semibold text-ink">
                            {Icon === undefined ? null : (
                                <Icon
                                    size={18}
                                    className={cx(
                                        "shrink-0",
                                        destructive ? toneClasses.danger.ink : toneClasses.accent.ink,
                                    )}
                                />
                            )}
                            {title}
                        </RadixDialog.Title>
                        <RadixDialog.Description className="text-sm text-ink-soft">
                            {description}
                        </RadixDialog.Description>
                        {children}
                    </div>
                    <div className="flex justify-end gap-2 border-t border-hairline bg-panel px-4 py-2.5">
                        <RadixDialog.Close asChild>
                            <Button variant="ghost">{cancelLabel}</Button>
                        </RadixDialog.Close>
                        <Button variant={destructive ? "danger" : "primary"} busy={busy} onClick={onConfirm}>
                            {confirmLabel}
                        </Button>
                    </div>
                </RadixDialog.Content>
            </RadixDialog.Portal>
        </RadixDialog.Root>
    );
}
