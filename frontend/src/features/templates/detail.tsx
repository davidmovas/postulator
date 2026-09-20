import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useTemplate } from "../../data/hooks/templates.js";
import type { Template } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Button,
    DeleteIcon,
    Dialog,
    IconButton,
    Menu,
    MoreHorizIcon,
    SectionLabel,
    SmartToyIcon,
    StatusBadge,
    VerifiedIcon,
} from "../../ui/index.js";
import { pageKindLabel, roleLabel, scopeLabel, scopeTone, stepLabel } from "./labels.js";
import { layerAt, patchObject, paths } from "./patch.js";
import { LayerBadge } from "./provenance.js";
import { sentencesOf } from "./sentences.js";
import { draftOf } from "./spec.js";

export interface TemplateDetailProps {
    template: Template;
    isDefault: boolean;
    duplicating: boolean;
    settingDefault: boolean;
    deleting: boolean;
    onOpen: () => void;
    onDuplicate: () => void;
    onSetDefault: () => void;
    onDelete: () => void;
    onAskAgent: () => void;
}

export function TemplateDetail({
    template,
    isDefault,
    duplicating,
    settingDefault,
    deleting,
    onOpen,
    onDuplicate,
    onSetDefault,
    onDelete,
    onAskAgent,
}: TemplateDetailProps): ReactElement {
    const detail = useTemplate(template.id);
    const [confirming, setConfirming] = useState(false);
    const draft = useMemo(() => draftOf(template.spec), [template.spec]);
    const overrides = useMemo(() => detail.data?.overrides ?? [], [detail.data]);
    const sitePatch = useMemo(
        () => patchObject(overrides.find((held) => held.scope === "site")?.patch),
        [overrides],
    );
    const changes = useMemo(
        () => (sitePatch === null ? [] : sentencesOf(draft, sitePatch)),
        [draft, sitePatch],
    );
    const enabled = draft.recipe.filter((step) => step.enabled);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-10 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                <div className="flex min-w-0 flex-col">
                    <span className="truncate text-sm font-semibold text-ink">{template.name}</span>
                    <span
                        className="truncate text-2xs text-ink-faint"
                        title={absoluteTime(template.updatedAt)}
                    >
                        {relativeTime(template.updatedAt)}
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
                        { key: "agent", label: copy.templates.askAgent, icon: SmartToyIcon, onSelect: onAskAgent },
                        {
                            key: "delete",
                            label: copy.templates.editor.deleteTemplate,
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
                    <StatusBadge tone={scopeTone(template.scope)} dot={false}>
                        {scopeLabel(template.scope)}
                    </StatusBadge>
                    <StatusBadge tone="muted" dot={false}>
                        {pageKindLabel(template.pageKind)}
                    </StatusBadge>
                    <StatusBadge tone="muted" dot={false}>
                        {copy.templates.versionLabel(template.version)}
                    </StatusBadge>
                    {isDefault ? (
                        <StatusBadge tone="accent" icon={VerifiedIcon}>
                            {copy.templates.siteDefault}
                        </StatusBadge>
                    ) : null}
                </div>

                <section className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between gap-2">
                        <SectionLabel>{copy.templates.detail.sections}</SectionLabel>
                        <LayerBadge layer={layerAt(paths.sections, sitePatch, null)} />
                    </div>
                    {draft.sections.length === 0 ? (
                        <p className="text-xs text-ink-faint">{copy.templates.detail.noSections}</p>
                    ) : (
                        <ol className="flex flex-col">
                            {draft.sections.map((section, index) => (
                                <li
                                    key={index}
                                    className="flex h-6 items-center justify-between gap-3 text-xs text-ink-soft"
                                >
                                    <span className="min-w-0 truncate">
                                        {section.heading === ""
                                            ? copy.templates.sections.untitled
                                            : section.heading}
                                    </span>
                                    <span className="shrink-0 font-mono text-2xs text-ink-faint">
                                        {copy.templates.targetWords(section.targetWords)}
                                    </span>
                                </li>
                            ))}
                        </ol>
                    )}
                </section>

                <section className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between gap-2">
                        <SectionLabel>{copy.templates.detail.models}</SectionLabel>
                        <LayerBadge layer={layerAt(paths.profiles, sitePatch, null)} />
                    </div>
                    {draft.profiles.length === 0 ? (
                        <p className="text-xs text-ink-faint">{copy.templates.detail.noModels}</p>
                    ) : (
                        <dl className="flex flex-col">
                            {draft.profiles.map((profile) => (
                                <div
                                    key={profile.role}
                                    className="flex h-5 items-center justify-between gap-3 text-xs"
                                >
                                    <dt className="shrink-0 text-ink-dim">{roleLabel(profile.role)}</dt>
                                    <dd className="min-w-0 truncate font-mono text-2xs text-ink-soft">
                                        {profile.model}
                                    </dd>
                                </div>
                            ))}
                        </dl>
                    )}
                </section>

                <section className="flex flex-col gap-1.5">
                    <div className="flex items-center justify-between gap-2">
                        <SectionLabel>{copy.templates.detail.recipe}</SectionLabel>
                        <LayerBadge layer={layerAt(paths.recipe, sitePatch, null)} />
                    </div>
                    <div className="flex flex-wrap gap-1">
                        {enabled.map((step) => (
                            <span
                                key={step.name}
                                className="inline-flex h-5 items-center rounded-sm border border-hairline bg-inset px-1.5 text-2xs text-ink-soft"
                            >
                                {stepLabel(step.name)}
                            </span>
                        ))}
                    </div>
                </section>

                {changes.length === 0 ? null : (
                    <section className="flex flex-col gap-1.5">
                        <SectionLabel>{copy.templates.detail.siteChanges}</SectionLabel>
                        <ul className="flex flex-col gap-0.5">
                            {changes.map((sentence) => (
                                <li key={sentence} className="text-xs text-ink-dim">
                                    {sentence}
                                </li>
                            ))}
                        </ul>
                    </section>
                )}
            </div>
            <footer className="flex shrink-0 flex-col gap-2 border-t border-hairline p-3">
                <Button variant="primary" data-open-editor={true} onClick={onOpen}>
                    {copy.templates.openEditor}
                </Button>
                <div className="grid grid-cols-2 gap-2">
                    <Button busy={duplicating} onClick={onDuplicate}>
                        {copy.templates.duplicate}
                    </Button>
                    <Button
                        busy={settingDefault}
                        disabled={isDefault}
                        title={isDefault ? copy.templates.alreadyDefault : undefined}
                        onClick={onSetDefault}
                    >
                        {copy.templates.setDefault}
                    </Button>
                </div>
            </footer>
            <Dialog
                open={confirming}
                onOpenChange={setConfirming}
                title={copy.templates.editor.deleteTitle}
                description={copy.templates.editor.deleteBody}
                confirmLabel={copy.templates.editor.deleteConfirm}
                cancelLabel={copy.templates.editor.cancel}
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
