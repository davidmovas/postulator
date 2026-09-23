import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";
import { Kbd } from "./kbd.js";

export interface MenuAction {
    key: string;
    label: string;
    icon?: IconComponent;
    active?: boolean;
    danger?: boolean;
    disabled?: boolean;
    shortcut?: readonly string[];
    onSelect: () => void;
}

export type MenuEntry =
    | MenuAction
    | { kind: "separator"; key: string }
    | { kind: "label"; key: string; label: string };

export type MenuAlign = "start" | "end";

export interface MenuProps {
    label: string;
    trigger: ReactNode;
    items: readonly MenuEntry[];
    align?: MenuAlign;
    width?: number;
}

function tone(item: MenuAction): string {
    if (item.danger === true) {
        return "text-danger data-[highlighted]:bg-danger-soft";
    }
    if (item.active === true) {
        return "text-accent data-[highlighted]:bg-inset";
    }
    return "text-ink-soft data-[highlighted]:bg-inset data-[highlighted]:text-ink";
}

export function Menu({ label, trigger, items, align = "end", width = 224 }: MenuProps): ReactElement {
    return (
        <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild={true}>{trigger}</DropdownMenu.Trigger>
            <DropdownMenu.Portal>
                <DropdownMenu.Content
                    align={align}
                    sideOffset={4}
                    aria-label={label}
                    style={{ width: `${width}px` }}
                    className="z-50 rounded-md border border-edge bg-raised p-1 data-[state=open]:animate-fade-in"
                >
                    {items.map((entry) => {
                        if ("kind" in entry) {
                            if (entry.kind === "separator") {
                                return (
                                    <DropdownMenu.Separator
                                        key={entry.key}
                                        className="my-1 h-px bg-hairline"
                                    />
                                );
                            }
                            return (
                                <DropdownMenu.Label
                                    key={entry.key}
                                    className="px-2 py-1 text-2xs font-semibold tracking-label text-ink-faint uppercase"
                                >
                                    {entry.label}
                                </DropdownMenu.Label>
                            );
                        }
                        const Icon = entry.icon;
                        return (
                            <DropdownMenu.Item
                                key={entry.key}
                                disabled={entry.disabled}
                                onSelect={entry.onSelect}
                                className={cx(
                                    "flex h-7 cursor-pointer items-center gap-2 rounded-sm px-2 text-xs outline-none",
                                    "data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50",
                                    tone(entry),
                                )}
                            >
                                {Icon === undefined ? null : <Icon size={14} className="shrink-0" />}
                                <span className="min-w-0 flex-1 truncate">{entry.label}</span>
                                {entry.shortcut === undefined ? null : <Kbd keys={entry.shortcut} />}
                            </DropdownMenu.Item>
                        );
                    })}
                </DropdownMenu.Content>
            </DropdownMenu.Portal>
        </DropdownMenu.Root>
    );
}
