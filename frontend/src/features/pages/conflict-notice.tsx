import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { isOneOf } from "../../generated/vocab.js";
import { ArrowRightAltIcon, cx, toneClasses, WarningIcon } from "../../ui/index.js";
import { cannibalizationReasons, conflictOf } from "./conflict.js";

function reasonOf(reason: string): string {
    return isOneOf(cannibalizationReasons, reason)
        ? copy.pages.conflict.reason[reason]
        : copy.pages.conflict.unknownReason;
}

export interface ConflictNoticeProps {
    thrown: unknown;
    siteId: string;
    search: string;
    className?: string;
}

export function ConflictNotice({ thrown, siteId, search, className }: ConflictNoticeProps): ReactElement | null {
    const conflict = conflictOf(thrown);
    if (conflict === null) {
        return null;
    }

    const heading =
        conflict.kind === "cannibalization" ? copy.pages.conflict.title : copy.pages.conflict.descendantsTitle;
    const body =
        conflict.kind === "cannibalization"
            ? copy.pages.conflict.body
            : copy.pages.conflict.descendants(conflict.descendants);

    return (
        <div
            role="note"
            className={cx(
                "flex gap-3 rounded-lg border p-3",
                toneClasses.warn.soft,
                toneClasses.warn.border,
                className,
            )}
        >
            <WarningIcon size={19} className={cx("mt-px shrink-0", toneClasses.warn.ink)} />
            <div className="flex min-w-0 flex-col gap-1.5">
                <p className={cx("text-sm font-semibold", toneClasses.warn.ink)}>{heading}</p>
                <p className="text-xs text-ink-soft">{body}</p>
                {conflict.kind === "cannibalization" ? (
                    <ul className="mt-0.5 flex flex-col gap-1">
                        {conflict.offenders.map((offender) => (
                            <li
                                key={`${offender.pageId}:${offender.path}:${offender.reason}`}
                                className="flex min-w-0 items-center gap-2 text-xs"
                            >
                                <ArrowRightAltIcon size={13} className="shrink-0 text-ink-faint" />
                                {offender.pageId === "" ? (
                                    <span className="truncate font-mono text-ink">{offender.path}</span>
                                ) : (
                                    <Link
                                        to={`/s/${siteId}/pages/${offender.pageId}${search}`}
                                        className="truncate font-mono"
                                    >
                                        {offender.path}
                                    </Link>
                                )}
                                <span className="shrink-0 text-ink-dim">{reasonOf(offender.reason)}</span>
                            </li>
                        ))}
                    </ul>
                ) : null}
            </div>
        </div>
    );
}
