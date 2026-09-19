import * as RadixTabs from "@radix-ui/react-tabs";
import type { ReactElement, ReactNode } from "react";

import { CountBadge } from "./badge.js";
import { cx } from "./cx.js";
import type { Tone } from "./tone.js";

export interface TabDefinition<T extends string> {
    value: T;
    label: string;
    count?: number;
    countTone?: Tone;
    disabled?: boolean;
}

export interface TabsProps<T extends string> {
    value: T;
    onValueChange: (value: T) => void;
    tabs: readonly TabDefinition<T>[];
    children: ReactNode;
    label: string;
    className?: string;
}

export function Tabs<T extends string>({
    value,
    onValueChange,
    tabs,
    children,
    label,
    className,
}: TabsProps<T>): ReactElement {
    return (
        <RadixTabs.Root
            value={value}
            onValueChange={(next) => {
                onValueChange(next as T);
            }}
            className={cx("flex min-h-0 min-w-0 flex-col", className)}
        >
            <RadixTabs.List
                aria-label={label}
                className="flex shrink-0 gap-0.5 border-b border-hairline px-1"
            >
                {tabs.map((tab) => (
                    <RadixTabs.Trigger
                        key={tab.value}
                        value={tab.value}
                        disabled={tab.disabled}
                        className={cx(
                            "inline-flex h-7 items-center gap-1.5 px-2.5 text-sm font-medium text-ink-dim",
                            "transition-colors duration-100 ease-out hover:text-ink disabled:cursor-not-allowed disabled:opacity-50",
                            "data-[state=active]:font-semibold data-[state=active]:text-ink",
                            "data-[state=active]:shadow-[inset_0_-2px_0_var(--color-accent)]",
                        )}
                    >
                        {tab.label}
                        {tab.count === undefined ? null : (
                            <CountBadge tone={tab.countTone ?? "muted"} count={tab.count} />
                        )}
                    </RadixTabs.Trigger>
                ))}
            </RadixTabs.List>
            {children}
        </RadixTabs.Root>
    );
}

export interface TabPanelProps<T extends string> {
    value: T;
    children: ReactNode;
    className?: string;
}

export function TabPanel<T extends string>({ value, children, className }: TabPanelProps<T>): ReactElement {
    return (
        <RadixTabs.Content value={value} className={cx("min-h-0 flex-1 overflow-auto outline-none", className)}>
            {children}
        </RadixTabs.Content>
    );
}
