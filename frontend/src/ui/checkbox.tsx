import type { InputHTMLAttributes, ReactElement } from "react";

import { cx } from "./cx.js";
import { CheckIcon, RemoveIcon } from "./icons/index.js";

export interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "children"> {
    label?: string;
    indeterminate?: boolean;
}

export function Checkbox({
    label,
    indeterminate = false,
    checked = false,
    disabled = false,
    className,
    ...rest
}: CheckboxProps): ReactElement {
    const marked = checked || indeterminate;
    return (
        <label
            className={cx(
                "inline-flex items-center gap-2 text-sm select-none",
                disabled ? "cursor-not-allowed text-ink-faint" : "cursor-pointer text-ink",
                className,
            )}
        >
            <input
                type="checkbox"
                checked={checked}
                disabled={disabled}
                aria-checked={indeterminate ? "mixed" : checked}
                className="peer sr-only"
                {...rest}
            />
            <span
                aria-hidden={true}
                className={cx(
                    "inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border transition-colors duration-100 ease-out",
                    "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent",
                    marked ? "border-accent bg-accent text-on-accent" : "border-edge bg-inset",
                    disabled && "opacity-60",
                )}
            >
                {indeterminate ? <RemoveIcon size={12} /> : checked ? <CheckIcon size={12} /> : null}
            </span>
            {label === undefined ? null : label}
        </label>
    );
}
