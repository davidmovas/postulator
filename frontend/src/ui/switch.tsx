import type { InputHTMLAttributes, ReactElement } from "react";

import { cx } from "./cx.js";

export interface SwitchProps extends Omit<InputHTMLAttributes<HTMLInputElement>, "type" | "children"> {
    label?: string;
    tone?: "accent" | "danger";
}

export function Switch({
    label,
    tone = "accent",
    checked = false,
    disabled = false,
    className,
    ...rest
}: SwitchProps): ReactElement {
    return (
        <label
            className={cx(
                "inline-flex items-center justify-between gap-3 text-sm select-none",
                disabled ? "cursor-not-allowed text-ink-faint" : "cursor-pointer text-ink",
                className,
            )}
        >
            {label === undefined ? null : <span>{label}</span>}
            <input type="checkbox" role="switch" checked={checked} disabled={disabled} className="peer sr-only" {...rest} />
            <span
                aria-hidden={true}
                className={cx(
                    "relative inline-block h-5 w-8 shrink-0 rounded-full border transition-colors duration-100 ease-out",
                    "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent",
                    checked
                        ? tone === "danger"
                            ? "border-transparent bg-danger"
                            : "border-transparent bg-accent"
                        : "border-hairline bg-raised-strong",
                    disabled && "opacity-60",
                )}
            >
                <span
                    className={cx(
                        "absolute top-0.5 h-4 w-4 rounded-full transition-[left] duration-150 ease-swift",
                        checked
                            ? tone === "danger"
                                ? "left-3.5 bg-on-danger"
                                : "left-3.5 bg-on-accent"
                            : "left-0.5 bg-ink-dim",
                    )}
                />
            </span>
        </label>
    );
}
