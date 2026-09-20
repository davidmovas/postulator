import type { KeyboardEvent, ReactElement, ReactNode } from "react";
import { useRef } from "react";
import { NavLink } from "react-router";

import { CountBadge } from "./badge.js";
import { cx } from "./cx.js";
import type { Tone } from "./tone.js";

export interface TabItem<T extends string = string> {
    key: T;
    label: string;
    to?: string;
    count?: number;
    countTone?: Tone;
    disabled?: boolean;
}

export interface TabsProps<T extends string = string> {
    label: string;
    items: readonly TabItem<T>[];
    value?: T;
    onValueChange?: (key: T) => void;
}

const stripBase = "flex h-full min-h-7 shrink-0 items-stretch gap-0.5";
const triggerBase =
    "inline-flex shrink-0 items-center gap-1.5 px-2.5 text-sm font-medium whitespace-nowrap transition-colors duration-100 ease-out";
const triggerActive = "font-semibold text-ink shadow-[inset_0_-2px_0_var(--color-accent)]";
const triggerIdle = "text-ink-dim hover:text-ink";
const triggerBlocked = "cursor-not-allowed text-ink-faint opacity-50";

function Count({ item }: { item: TabItem }): ReactElement | null {
    if (item.count === undefined) {
        return null;
    }
    return <CountBadge tone={item.countTone ?? "muted"} count={item.count} />;
}

function step(items: readonly TabItem<string>[], from: number, delta: number): number {
    const total = items.length;
    for (let move = 1; move <= total; move += 1) {
        const at = (from + delta * move + total * total) % total;
        if (items[at].disabled !== true) {
            return at;
        }
    }
    return from;
}

export function Tabs<T extends string = string>({ label, items, value, onValueChange }: TabsProps<T>): ReactElement {
    const triggers = useRef<(HTMLButtonElement | null)[]>([]);

    if (items.some((item) => item.to !== undefined)) {
        return (
            <nav aria-label={label} className={stripBase}>
                {items.map((item) => (
                    <NavLink
                        key={item.key}
                        to={item.to ?? ""}
                        className={({ isActive }) =>
                            cx(triggerBase, isActive ? triggerActive : triggerIdle)
                        }
                    >
                        {item.label}
                        <Count item={item} />
                    </NavLink>
                ))}
            </nav>
        );
    }

    const move = (event: KeyboardEvent<HTMLButtonElement>, from: number): void => {
        const delta = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
        const at =
            event.key === "Home"
                ? step(items, items.length - 1, 1)
                : event.key === "End"
                  ? step(items, 0, -1)
                  : delta === 0
                    ? -1
                    : step(items, from, delta);
        if (at < 0) {
            return;
        }
        event.preventDefault();
        triggers.current[at]?.focus();
        onValueChange?.(items[at].key);
    };

    return (
        <div role="tablist" aria-label={label} className={stripBase}>
            {items.map((item, index) => (
                <button
                    key={item.key}
                    ref={(node) => {
                        triggers.current[index] = node;
                    }}
                    type="button"
                    role="tab"
                    aria-selected={item.key === value}
                    tabIndex={item.key === value ? 0 : -1}
                    disabled={item.disabled}
                    onClick={() => {
                        onValueChange?.(item.key);
                    }}
                    onKeyDown={(event) => {
                        move(event, index);
                    }}
                    className={cx(
                        triggerBase,
                        item.disabled === true
                            ? triggerBlocked
                            : item.key === value
                              ? triggerActive
                              : triggerIdle,
                    )}
                >
                    {item.label}
                    <Count item={item} />
                </button>
            ))}
        </div>
    );
}

export interface TabPanelProps {
    label: string;
    active: boolean;
    children: ReactNode;
}

export function TabPanel({ label, active, children }: TabPanelProps): ReactElement | null {
    if (!active) {
        return null;
    }
    return (
        <div role="tabpanel" aria-label={label} className="flex min-h-0 flex-1 flex-col overflow-auto outline-none">
            {children}
        </div>
    );
}
