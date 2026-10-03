import type { ReactElement } from "react";

import { cx } from "./cx.js";
import { CategoryIcon } from "./icons/index.js";
import { toneClasses } from "./tone.js";

export type CategoryState = "onSite" | "onPublish" | "needsPlugin" | "becomes";

export interface CategoryTrailItem {
    key: string;
    name: string;
    state: CategoryState;
    hint: string;
}

export interface CategoryTrailProps {
    items: readonly CategoryTrailItem[];
    label: string;
    compact?: boolean;
    className?: string;
}

const arrow = "›";

const stateClasses: Readonly<Record<CategoryState, string>> = {
    onSite: "border-hairline bg-inset text-ink-soft",
    onPublish: "border-dashed border-edge text-ink-dim",
    needsPlugin: cx(toneClasses.warn.soft, toneClasses.warn.border, toneClasses.warn.ink),
    becomes: cx("border-dashed", toneClasses.info.border, toneClasses.info.ink),
};

function wholeTrail(items: readonly CategoryTrailItem[]): string {
    return [items.map((item) => item.name).join(` ${arrow} `), ...items.map((item) => `${item.name}: ${item.hint}`)].join(
        "\n",
    );
}

export function CategoryTrail({ items, label, compact = false, className }: CategoryTrailProps): ReactElement | null {
    if (items.length === 0) {
        return null;
    }
    return (
        <ol
            aria-label={label}
            data-category-trail={true}
            title={compact ? wholeTrail(items) : undefined}
            className={cx(
                "flex min-w-0 items-center gap-1 font-sans",
                compact ? "flex-nowrap overflow-hidden whitespace-nowrap" : "flex-wrap",
                className,
            )}
        >
            {items.map((item, at) => (
                <li
                    key={item.key}
                    data-category-state={item.state}
                    title={compact ? undefined : item.hint}
                    className="flex min-w-0 items-center gap-1"
                >
                    {at === 0 ? null : (
                        <span aria-hidden={true} className="shrink-0 text-ink-faint">
                            {arrow}
                        </span>
                    )}
                    <span
                        className={cx(
                            "inline-flex min-w-0 items-center gap-1 rounded-sm border font-medium",
                            compact ? "h-4 px-1 text-2xs" : "h-5 px-1.5 text-xs",
                            stateClasses[item.state],
                        )}
                    >
                        <CategoryIcon size={compact ? 11 : 12} className="shrink-0" />
                        <span className="truncate">{item.name}</span>
                    </span>
                </li>
            ))}
        </ol>
    );
}
