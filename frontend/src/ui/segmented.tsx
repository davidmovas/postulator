import type { ReactElement } from "react";

import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";

export type SegmentedSize = "md" | "sm";

export interface SegmentedOption<T extends string> {
    value: T;
    label: string;
    icon?: IconComponent;
    disabled?: boolean;
    title?: string;
}

export interface SegmentedProps<T extends string> {
    label: string;
    value: T;
    options: readonly SegmentedOption<T>[];
    onValueChange: (value: T) => void;
    size?: SegmentedSize;
    iconOnly?: boolean;
}

const sizeClasses: Readonly<Record<SegmentedSize, string>> = {
    md: "h-6 gap-1.5 px-2.5 text-xs",
    sm: "h-5 gap-1 px-1.5 text-2xs",
};

const iconSize: Readonly<Record<SegmentedSize, number>> = { md: 14, sm: 13 };

export function Segmented<T extends string>({
    label,
    value,
    options,
    onValueChange,
    size = "md",
    iconOnly = false,
}: SegmentedProps<T>): ReactElement {
    return (
        <div
            role="radiogroup"
            aria-label={label}
            className="inline-flex shrink-0 items-center gap-0.5 rounded-md bg-inset p-0.5"
        >
            {options.map((option) => {
                const Icon = option.icon;
                const picked = option.value === value;
                return (
                    <button
                        key={option.value}
                        type="button"
                        role="radio"
                        aria-checked={picked}
                        aria-label={iconOnly ? option.label : undefined}
                        title={option.title ?? (iconOnly ? option.label : undefined)}
                        disabled={option.disabled}
                        onClick={() => {
                            onValueChange(option.value);
                        }}
                        className={cx(
                            "inline-flex items-center justify-center rounded-md font-medium whitespace-nowrap",
                            "transition-colors duration-100 ease-out disabled:cursor-not-allowed disabled:opacity-50",
                            sizeClasses[size],
                            picked ? "bg-raised text-ink" : "text-ink-dim enabled:hover:text-ink",
                        )}
                    >
                        {Icon === undefined ? null : <Icon size={iconSize[size]} />}
                        {iconOnly && Icon !== undefined ? null : option.label}
                    </button>
                );
            })}
        </div>
    );
}
