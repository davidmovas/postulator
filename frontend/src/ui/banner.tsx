import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import { CheckCircleIcon, ErrorIcon, InfoIcon, RemoveIcon, WarningIcon } from "./icons/index.js";
import type { IconComponent } from "./icons/index.js";
import type { Tone } from "./tone.js";
import { toneClasses } from "./tone.js";

const toneIcon: Readonly<Record<Tone, IconComponent>> = {
    accent: InfoIcon,
    ok: CheckCircleIcon,
    warn: WarningIcon,
    danger: ErrorIcon,
    info: InfoIcon,
    muted: RemoveIcon,
};

export interface BannerProps {
    tone: Tone;
    title: string;
    body?: string;
    icon?: IconComponent;
    actions?: ReactNode;
    className?: string;
}

export function Banner({ tone, title, body, icon, actions, className }: BannerProps): ReactElement {
    const Icon = icon ?? toneIcon[tone];
    const classes = toneClasses[tone];
    return (
        <div
            role="note"
            className={cx("flex gap-3 rounded-lg border p-3", classes.soft, classes.border, className)}
        >
            <Icon size={19} className={cx("mt-px shrink-0", classes.ink)} />
            <div className="flex min-w-0 flex-col gap-1.5">
                <p className={cx("text-sm font-semibold", classes.ink)}>{title}</p>
                {body === undefined ? null : <p className="text-xs text-ink-soft">{body}</p>}
                {actions === undefined ? null : <div className="mt-0.5 flex flex-wrap gap-2">{actions}</div>}
            </div>
        </div>
    );
}
