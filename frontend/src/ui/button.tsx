import type { ButtonHTMLAttributes, ReactElement } from "react";

import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";
import { Spinner } from "./spinner.js";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "md" | "sm";

const variantClasses: Readonly<Record<ButtonVariant, string>> = {
    primary: "bg-accent text-on-accent border border-transparent enabled:hover:bg-accent-hover",
    secondary: "bg-raised text-ink border border-hairline enabled:hover:bg-raised-strong",
    ghost: "bg-transparent text-ink-soft border border-transparent enabled:hover:bg-inset enabled:hover:text-ink",
    danger: "bg-danger-soft text-danger border border-danger-border enabled:hover:bg-danger enabled:hover:text-on-danger",
};

const refusedClasses: Readonly<Record<ButtonVariant, string>> = {
    primary: "disabled:border-transparent disabled:bg-inset disabled:text-ink-faint",
    secondary: "disabled:border-hairline disabled:bg-transparent disabled:text-ink-faint",
    ghost: "disabled:border-transparent disabled:bg-transparent disabled:text-ink-faint",
    danger: "disabled:border-hairline disabled:bg-transparent disabled:text-ink-faint",
};

const sizeClasses: Readonly<Record<ButtonSize, string>> = {
    md: "h-7 px-3 text-sm",
    sm: "h-6 px-2 text-xs",
};

const iconSize: Readonly<Record<ButtonSize, number>> = { md: 16, sm: 14 };

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: ButtonVariant;
    size?: ButtonSize;
    icon?: IconComponent;
    busy?: boolean;
}

export function Button({
    variant = "secondary",
    size = "md",
    icon: Icon,
    busy = false,
    disabled = false,
    className,
    children,
    type = "button",
    ...rest
}: ButtonProps): ReactElement {
    const blocked = disabled || busy;
    return (
        <button
            type={type}
            disabled={blocked}
            aria-busy={busy || undefined}
            aria-disabled={disabled || undefined}
            className={cx(
                "inline-flex shrink-0 items-center justify-center gap-1 rounded-md font-semibold whitespace-nowrap transition-colors duration-100 ease-out",
                "disabled:cursor-default",
                variantClasses[variant],
                sizeClasses[size],
                busy ? "opacity-80" : refusedClasses[variant],
                className,
            )}
            {...rest}
        >
            {busy ? <Spinner size={iconSize[size] - 4} /> : Icon === undefined ? null : <Icon size={iconSize[size]} />}
            {children}
        </button>
    );
}

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
    icon: IconComponent;
    label: string;
    variant?: ButtonVariant;
    size?: ButtonSize;
}

export function IconButton({
    icon: Icon,
    label,
    variant = "secondary",
    size = "md",
    disabled = false,
    className,
    type = "button",
    ...rest
}: IconButtonProps): ReactElement {
    return (
        <button
            type={type}
            disabled={disabled}
            aria-label={label}
            title={label}
            className={cx(
                "inline-flex shrink-0 items-center justify-center rounded-md transition-colors duration-100 ease-out",
                "disabled:cursor-default",
                variantClasses[variant],
                refusedClasses[variant],
                size === "md" ? "h-7 w-7" : "h-6 w-6",
                className,
            )}
            {...rest}
        >
            <Icon size={iconSize[size] + 1} />
        </button>
    );
}
