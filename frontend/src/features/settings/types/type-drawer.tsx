import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import type { CatalogField, PageType } from "../../../data/types.js";
import { Banner, Button, DeleteIcon, Drawer, EditNoteIcon, SectionLabel, StatusBadge, toneClasses } from "../../../ui/index.js";
import { CustomFieldForm } from "./custom-field.js";
import { presetHint, presetIcon, presetLabel, presetTone } from "./labels.js";
import { aiWritableOf, lastOfPreset, orderedScopes, usualFor } from "./model.js";
import { ScopeCard } from "./scope-card.js";
import type { TemplateChoice } from "./type-dialogs.js";
import { DeleteTypeDialog, TypeFormDialog } from "./type-dialogs.js";

const said = copy.pageTypes.drawer;

export interface TypeDrawerProps {
    siteId: string;
    type: PageType;
    types: readonly PageType[];
    catalog: readonly CatalogField[];
    templates: readonly TemplateChoice[];
    templateName: string;
    onClose: () => void;
}

export function TypeDrawer({ siteId, type, types, catalog, templates, templateName, onClose }: TypeDrawerProps): ReactElement {
    const [editing, setEditing] = useState(false);
    const [deleting, setDeleting] = useState(false);
    const [showOthers, setShowOthers] = useState(false);
    const Icon = presetIcon(type.preset);
    const scopes = orderedScopes(type.scopes ?? []);
    const usual = scopes.filter((scope) => usualFor(scope, type.preset));
    const others = scopes.filter((scope) => !usualFor(scope, type.preset));
    const last = lastOfPreset(types, type);
    const role = presetLabel(type.preset);

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={type.name}
            description={presetHint(type.preset)}
            closeLabel={said.close}
            width={640}
        >
            <div className="flex flex-col gap-4 p-4" data-page-type-drawer={type.id}>
                <section className="flex flex-col gap-2" aria-label={said.identity}>
                    <SectionLabel>{said.identity}</SectionLabel>
                    <dl className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-x-3 gap-y-1.5 text-sm">
                        <dt className="text-ink-dim">{said.role}</dt>
                        <dd className="flex items-center gap-1.5 text-ink">
                            <Icon size={14} className={toneClasses[presetTone(type.preset)].ink} />
                            {role}
                            {type.writable ? null : (
                                <StatusBadge tone="muted" dot={false}>
                                    {copy.pageTypes.screen.readOnly}
                                </StatusBadge>
                            )}
                        </dd>
                        <dt className="text-ink-dim">{said.pages}</dt>
                        <dd className="font-mono text-xs text-ink-soft">{copy.pageTypes.screen.pages(type.pages)}</dd>
                        <dt className="text-ink-dim">{said.template}</dt>
                        <dd className="truncate text-ink-soft">{templateName}</dd>
                    </dl>
                    <div className="flex gap-2">
                        <Button
                            size="sm"
                            icon={EditNoteIcon}
                            data-type-edit={true}
                            onClick={() => {
                                setEditing(true);
                            }}
                        >
                            {said.edit}
                        </Button>
                        <Button
                            size="sm"
                            variant="ghost"
                            icon={DeleteIcon}
                            disabled={last}
                            title={last ? copy.pageTypes.remove.lastOfPreset : undefined}
                            data-type-delete={true}
                            onClick={() => {
                                setDeleting(true);
                            }}
                        >
                            {copy.pageTypes.remove.action}
                        </Button>
                    </div>
                </section>

                <section className="flex flex-col gap-2" aria-label={said.fields}>
                    <SectionLabel>{said.fields}</SectionLabel>
                    {type.writable ? (
                        <p className="text-2xs text-ink-faint">{said.fieldsBody}</p>
                    ) : (
                        <Banner tone="info" title={said.readOnly(role)} />
                    )}
                    <ul className="flex flex-col gap-1.5">
                        {usual.map((scope) => (
                            <ScopeCard
                                key={scope.field}
                                type={type}
                                scope={scope}
                                aiWritable={aiWritableOf(catalog, scope.field)}
                            />
                        ))}
                    </ul>
                    {others.length === 0 ? null : (
                        <>
                            <Button
                                size="sm"
                                variant="ghost"
                                className="self-start"
                                aria-expanded={showOthers}
                                onClick={() => {
                                    setShowOthers((held) => !held);
                                }}
                            >
                                {showOthers ? said.hideOthers : said.others(others.length)}
                            </Button>
                            {showOthers ? (
                                <ul className="flex flex-col gap-1.5">
                                    {others.map((scope) => (
                                        <ScopeCard
                                            key={scope.field}
                                            type={type}
                                            scope={scope}
                                            aiWritable={aiWritableOf(catalog, scope.field)}
                                        />
                                    ))}
                                </ul>
                            ) : null}
                        </>
                    )}
                </section>

                {type.writable ? <CustomFieldForm type={type} /> : null}
            </div>
            <TypeFormDialog
                open={editing}
                siteId={siteId}
                type={type}
                templates={templates}
                onOpenChange={setEditing}
            />
            <DeleteTypeDialog open={deleting} type={type} types={types} onOpenChange={setDeleting} onDeleted={onClose} />
        </Drawer>
    );
}
