import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import type { LinkPolicy } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Button,
    DeleteIcon,
    Dialog,
    EditNoteIcon,
    IconButton,
    Menu,
    MoreHorizIcon,
    SectionLabel,
    StatusBadge,
    VerifiedIcon,
} from "../../ui/index.js";
import { anchorLabel, decimal, flagLabel, scopeLabel, scopeTone } from "./labels.js";

interface RuleRow {
    label: string;
    value: string;
}

function rulesOf(policy: LinkPolicy): readonly RuleRow[] {
    return [
        { label: copy.templates.links.upDepth, value: String(policy.rules.upDepth) },
        { label: copy.templates.links.downLinks, value: flagLabel(policy.rules.downLinks) },
        { label: copy.templates.links.siblingMinWeight, value: decimal(policy.rules.siblingMinWeight) },
        { label: copy.templates.links.maxLinks, value: String(policy.rules.maxLinks) },
        { label: copy.templates.links.maxPerTarget, value: String(policy.rules.maxPerTarget) },
        {
            label: copy.templates.links.parentLinkWithinParagraphs,
            value: String(policy.rules.parentLinkWithinParagraphs),
        },
        { label: copy.templates.links.childrenSection, value: flagLabel(policy.rules.childrenSection) },
    ];
}

export interface PolicyDetailProps {
    policy: LinkPolicy;
    inForce: boolean;
    chosenHere: boolean;
    applying: boolean;
    deleting: boolean;
    onEdit: () => void;
    onUseHere: () => void;
    onDelete: () => void;
}

export function PolicyDetail({
    policy,
    inForce,
    chosenHere,
    applying,
    deleting,
    onEdit,
    onUseHere,
    onDelete,
}: PolicyDetailProps): ReactElement {
    const [confirming, setConfirming] = useState(false);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-10 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                <div className="flex min-w-0 flex-col">
                    <span className="truncate text-sm font-semibold text-ink">{policy.name}</span>
                    <span className="truncate text-2xs text-ink-faint" title={absoluteTime(policy.updatedAt)}>
                        {relativeTime(policy.updatedAt)}
                    </span>
                </div>
                <Menu
                    label={copy.templates.moreActions}
                    trigger={
                        <IconButton
                            icon={MoreHorizIcon}
                            label={copy.templates.moreActions}
                            variant="ghost"
                            size="sm"
                        />
                    }
                    items={[
                        {
                            key: "delete",
                            label: copy.policies.delete,
                            icon: DeleteIcon,
                            danger: true,
                            onSelect: () => {
                                setConfirming(true);
                            },
                        },
                    ]}
                />
            </header>
            <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-3">
                <div className="flex flex-wrap items-center gap-1.5">
                    <StatusBadge tone={scopeTone(policy.scope)} dot={false}>
                        {scopeLabel(policy.scope)}
                    </StatusBadge>
                    {inForce ? (
                        <StatusBadge tone="accent" icon={VerifiedIcon}>
                            {copy.policies.inForce}
                        </StatusBadge>
                    ) : null}
                </div>
                {inForce ? (
                    <p className="text-xs text-ink-dim">
                        {chosenHere ? copy.policies.effectiveFromSite : copy.policies.effectiveFallback}
                    </p>
                ) : null}
                <dl className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 text-xs">
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.forbidExternal}</dt>
                    <dd className="text-ink">{flagLabel(policy.forbidExternal)}</dd>
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.forbidSelf}</dt>
                    <dd className="text-ink">{flagLabel(policy.forbidSelf)}</dd>
                    <dt className="min-w-0 truncate text-ink-dim">{copy.policies.anchorStrategy}</dt>
                    <dd className="text-ink">{anchorLabel(policy.anchorStrategy)}</dd>
                </dl>
                <section className="flex flex-col gap-1.5">
                    <span className="w-fit cursor-help" title={copy.policies.rulesTooltip}>
                        <SectionLabel>{copy.policies.rulesTitle}</SectionLabel>
                    </span>
                    <dl className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 text-xs">
                        {rulesOf(policy).map((row) => (
                            <div key={row.label} className="contents">
                                <dt className="min-w-0 truncate text-ink-dim">{row.label}</dt>
                                <dd className="font-mono text-2xs text-ink-soft">{row.value}</dd>
                            </div>
                        ))}
                    </dl>
                </section>
            </div>
            <footer className="flex shrink-0 flex-col gap-2 border-t border-hairline p-3">
                <Button variant="primary" icon={EditNoteIcon} data-policy-edit={true} onClick={onEdit}>
                    {copy.policies.edit}
                </Button>
                <Button
                    busy={applying}
                    disabled={chosenHere}
                    title={chosenHere ? copy.policies.alreadyInForce : undefined}
                    onClick={onUseHere}
                >
                    {copy.policies.useHere}
                </Button>
            </footer>
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={copy.policies.deleteTitle}
                description={copy.policies.deleteBody}
                confirmLabel={copy.policies.deleteConfirm}
                cancelLabel={copy.policies.create.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={deleting}
                onConfirm={() => {
                    setConfirming(false);
                    onDelete();
                }}
            />
        </div>
    );
}
