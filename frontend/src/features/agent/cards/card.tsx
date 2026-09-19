import type { KeyboardEvent, ReactElement, ReactNode } from "react";
import { useEffect, useRef } from "react";

import { copy } from "../../../copy/index.js";
import type { Timestamp } from "../../../data/wire.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import { Button, cx, GavelIcon, ShieldIcon, StatusBadge, toneClasses, WarningIcon } from "../../../ui/index.js";
import { familyOf, verbOf } from "../conversation/model/tools.js";
import { familyIcon, familyLabel, riskLabel, riskTone } from "../labels.js";

export type CardBusy = "approve" | "reject" | null;

export interface ConfirmationCardProps {
    tool: string;
    risk: string;
    status: string;
    createdAt: Timestamp;
    outcome: string | null;
    lines?: ReactNode;
    busy: CardBusy;
    focus?: boolean;
    onApprove?: () => void;
    onReject?: () => void;
    className?: string;
}

export function ConfirmationCard({
    tool,
    risk,
    status,
    createdAt,
    outcome,
    lines,
    busy,
    focus = false,
    onApprove,
    onReject,
    className,
}: ConfirmationCardProps): ReactElement {
    const root = useRef<HTMLDivElement>(null);
    const dangerous = risk === "dangerous";
    const pending = status === "pending";
    const tone = riskTone(risk);
    const classes = toneClasses[tone];
    const family = familyOf(tool);
    const FamilyIcon = familyIcon(family);
    const RiskIcon = dangerous ? GavelIcon : ShieldIcon;

    useEffect(() => {
        if (focus && pending) {
            root.current?.focus();
        }
    }, [focus, pending]);

    const keyed = (event: KeyboardEvent<HTMLDivElement>): void => {
        if (!pending || busy !== null) {
            return;
        }
        if (event.key === "Enter" && (event.ctrlKey || event.metaKey) && onApprove !== undefined) {
            event.preventDefault();
            event.stopPropagation();
            onApprove();
        } else if (event.key === "Escape" && onReject !== undefined) {
            event.preventDefault();
            event.stopPropagation();
            onReject();
        }
    };

    return (
        <div
            ref={root}
            tabIndex={pending ? 0 : -1}
            role="group"
            aria-label={copy.agent.confirmTitle}
            onKeyDown={keyed}
            className={cx(
                "flex flex-col overflow-hidden rounded-lg border bg-panel outline-none",
                pending ? classes.border : "border-hairline",
                dangerous && pending && "border-2",
                "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent",
                className,
            )}
        >
            <div className={cx("flex items-center gap-2 border-b px-3 py-1.5", pending ? classes.border : "border-hairline", pending && classes.soft)}>
                <StatusBadge tone={pending ? tone : "muted"} icon={RiskIcon}>
                    {riskLabel(risk)}
                </StatusBadge>
                <span className={cx("min-w-0 flex-1 truncate text-xs font-semibold", pending ? classes.ink : "text-ink-dim")}>
                    {dangerous ? copy.agent.card.kickerDangerous : copy.agent.card.kickerWrite}
                </span>
                <span className="shrink-0 font-mono text-2xs text-ink-faint" title={absoluteTime(createdAt)}>
                    {copy.agent.card.requested(relativeTime(createdAt))}
                </span>
            </div>
            <div className="flex flex-col gap-2.5 p-3">
                <div className="flex items-center gap-2">
                    <FamilyIcon size={16} className={cx("shrink-0", pending ? classes.ink : "text-ink-faint")} />
                    <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{familyLabel(family)}</span>
                    <span className="text-sm font-semibold text-ink">{verbOf(tool)}</span>
                </div>
                {lines === undefined ? null : <div className="flex flex-col gap-1.5">{lines}</div>}
                {dangerous && pending ? (
                    <div className={cx("flex items-start gap-2 rounded-md border px-2.5 py-2", classes.soft, classes.border)}>
                        <WarningIcon size={15} className={cx("mt-px shrink-0", classes.ink)} />
                        <span className={cx("text-xs font-medium", classes.ink)}>{copy.agent.card.dangerousWarn}</span>
                    </div>
                ) : null}
                {pending ? (
                    <div className="flex items-center gap-2">
                        <Button
                            variant={dangerous ? "danger" : "primary"}
                            className="flex-1"
                            busy={busy === "approve"}
                            disabled={busy !== null || onApprove === undefined}
                            onClick={onApprove}
                        >
                            {busy === "approve" ? copy.agent.card.approving : copy.agent.approve}
                        </Button>
                        <Button
                            variant="secondary"
                            busy={busy === "reject"}
                            disabled={busy !== null || onReject === undefined}
                            onClick={onReject}
                        >
                            {busy === "reject" ? copy.agent.card.rejecting : copy.agent.reject}
                        </Button>
                    </div>
                ) : null}
                <div className="flex items-center justify-between gap-2 font-mono text-2xs text-ink-faint">
                    <span>{pending ? copy.agent.card.hints : outcome}</span>
                    <span className="truncate">{copy.agent.card.tool(tool)}</span>
                </div>
            </div>
        </div>
    );
}
