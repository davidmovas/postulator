import type { ReactElement, ReactNode } from "react";

import { IconButton } from "./button.js";
import { cx } from "./cx.js";
import { CheckCircleIcon, CloseIcon, ErrorIcon, InfoIcon, WarningIcon } from "./icons/index.js";
import type { IconComponent } from "./icons/index.js";
import type { Tone } from "./tone.js";
import { toneClasses } from "./tone.js";

const toneIcon: Readonly<Record<Tone, IconComponent>> = {
    accent: InfoIcon,
    ok: CheckCircleIcon,
    warn: WarningIcon,
    danger: ErrorIcon,
    info: InfoIcon,
    muted: InfoIcon,
};

export interface ToastProps {
    tone: Tone;
    message: string;
    dismissLabel: string;
    onDismiss: () => void;
    action?: ReactNode;
    className?: string;
}

export function Toast({ tone, message, dismissLabel, onDismiss, action, className }: ToastProps): ReactElement {
    const Icon = toneIcon[tone];
    return (
        <div
            role="status"
            className={cx(
                "flex items-center gap-2 rounded-lg border border-edge bg-raised px-3 py-2",
                "animate-slide-in-right",
                className,
            )}
        >
            <Icon size={17} className={cx("shrink-0", toneClasses[tone].ink)} />
            <span className="min-w-0 flex-1 text-sm font-medium text-ink">{message}</span>
            {action}
            <IconButton icon={CloseIcon} label={dismissLabel} variant="ghost" size="sm" onClick={onDismiss} />
        </div>
    );
}

export interface ToastRegionProps {
    children: ReactNode;
    className?: string;
}

export function ToastRegion({ children, className }: ToastRegionProps): ReactElement {
    return (
        <div
            aria-live="polite"
            className={cx(
                "pointer-events-none fixed right-4 bottom-10 z-50 flex w-80 flex-col gap-2 [&>*]:pointer-events-auto",
                className,
            )}
        >
            {children}
        </div>
    );
}
