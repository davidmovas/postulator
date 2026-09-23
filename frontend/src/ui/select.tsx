import * as RadixSelect from "@radix-ui/react-select";
import type { ReactElement } from "react";

import { cx } from "./cx.js";
import type { ControlSize } from "./field.js";
import { CheckIcon, KeyboardArrowDownIcon, KeyboardArrowUpIcon } from "./icons/index.js";

const triggerSizes: Readonly<Record<ControlSize, string>> = {
    md: "h-7 px-2.5 text-sm",
    sm: "h-6 px-2 text-xs",
};

const chevronSizes: Readonly<Record<ControlSize, number>> = { md: 18, sm: 14 };

export interface SelectOption<T extends string> {
    value: T;
    label: string;
    disabled?: boolean;
}

export interface SelectProps<T extends string> {
    value: T | null;
    options: readonly SelectOption<T>[];
    onValueChange: (value: T) => void;
    id?: string;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    size?: ControlSize;
    quiet?: boolean;
    "aria-label"?: string;
    "aria-describedby"?: string;
}

export function Select<T extends string>({
    value,
    options,
    onValueChange,
    id,
    placeholder,
    disabled = false,
    invalid = false,
    size = "md",
    quiet = false,
    ...aria
}: SelectProps<T>): ReactElement {
    return (
        <RadixSelect.Root
            value={value ?? ""}
            onValueChange={(next) => {
                onValueChange(next as T);
            }}
            disabled={disabled}
        >
            <RadixSelect.Trigger
                id={id}
                aria-invalid={invalid || undefined}
                className={cx(
                    "inline-flex w-full items-center justify-between gap-2 rounded-md border text-ink",
                    triggerSizes[size],
                    "transition-colors duration-100 ease-out data-[placeholder]:text-ink-faint",
                    "disabled:cursor-not-allowed disabled:opacity-70",
                    quiet
                        ? "border-transparent bg-transparent text-ink-dim hover:bg-inset hover:text-ink data-[state=open]:border-edge data-[state=open]:bg-inset"
                        : "bg-inset",
                    invalid ? "border-danger" : quiet ? "" : "border-hairline hover:border-edge",
                )}
                {...aria}
            >
                <span className="min-w-0 flex-1 truncate text-left">
                    <RadixSelect.Value placeholder={placeholder} />
                </span>
                <RadixSelect.Icon asChild>
                    <KeyboardArrowDownIcon size={chevronSizes[size]} className="shrink-0 text-ink-faint" />
                </RadixSelect.Icon>
            </RadixSelect.Trigger>
            <RadixSelect.Portal>
                <RadixSelect.Content
                    position="popper"
                    sideOffset={4}
                    className="z-50 max-h-72 min-w-(--radix-select-trigger-width) overflow-hidden rounded-lg border border-edge bg-panel"
                >
                    <RadixSelect.ScrollUpButton className="flex h-5 items-center justify-center text-ink-dim">
                        <KeyboardArrowUpIcon size={16} />
                    </RadixSelect.ScrollUpButton>
                    <RadixSelect.Viewport className="p-1">
                        {options.map((option) => (
                            <RadixSelect.Item
                                key={option.value}
                                value={option.value}
                                disabled={option.disabled}
                                className={cx(
                                    "flex h-7 cursor-default items-center gap-2 rounded-sm px-2 text-sm text-ink-soft outline-none",
                                    "data-highlighted:bg-raised data-highlighted:text-ink",
                                    "data-[state=checked]:text-ink data-disabled:opacity-50",
                                )}
                            >
                                <RadixSelect.ItemText>{option.label}</RadixSelect.ItemText>
                                <RadixSelect.ItemIndicator asChild>
                                    <CheckIcon size={14} className="ml-auto text-accent" />
                                </RadixSelect.ItemIndicator>
                            </RadixSelect.Item>
                        ))}
                    </RadixSelect.Viewport>
                    <RadixSelect.ScrollDownButton className="flex h-5 items-center justify-center text-ink-dim">
                        <KeyboardArrowDownIcon size={16} />
                    </RadixSelect.ScrollDownButton>
                </RadixSelect.Content>
            </RadixSelect.Portal>
        </RadixSelect.Root>
    );
}
