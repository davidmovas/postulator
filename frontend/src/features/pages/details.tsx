import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useUpdatePage } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { pageStatuses, pageWpTypes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Banner, Button, Field, Input, Select, SyncProblemIcon, Textarea } from "../../ui/index.js";
import { ConflictNotice } from "./conflict-notice.js";

const wpTypeOptions: readonly SelectOption<string>[] = pageWpTypes.map((value) => ({ value, label: value }));
const statusOptions: readonly SelectOption<string>[] = pageStatuses.map((value) => ({ value, label: value }));

interface Draft {
    path: string;
    title: string;
    h1: string;
    metaTitle: string;
    metaDescription: string;
    canonical: string;
    wpType: string;
    status: string;
}

function draftOf(page: Page): Draft {
    return {
        path: page.path,
        title: page.title,
        h1: page.h1,
        metaTitle: page.metaTitle,
        metaDescription: page.metaDescription,
        canonical: page.canonical,
        wpType: page.wpType,
        status: page.status,
    };
}

function fieldErrorOf(thrown: unknown, field: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "field" && reaction.field === field ? reaction.message : null;
}

function formErrorOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "form" ? reaction.message : null;
}

function DriftNotice({ page }: { page: Page }): ReactElement {
    return (
        <Banner
            tone="warn"
            icon={SyncProblemIcon}
            title={copy.pages.drift.title}
            body={
                <div className="flex flex-col gap-1">
                    <p>{copy.pages.drift.decision}</p>
                    <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 font-mono text-2xs text-ink-faint">
                        <dt>{copy.pages.drift.lastWritten}</dt>
                        <dd title={absoluteTime(page.lastSyncedAt)}>{relativeTime(page.lastSyncedAt)}</dd>
                        <dt>{copy.pages.drift.wpModified}</dt>
                        <dd title={absoluteTime(page.wpModifiedAt)}>{relativeTime(page.wpModifiedAt)}</dd>
                    </dl>
                </div>
            }
        />
    );
}

export interface PageDetailsProps {
    page: Page;
    siteId: string;
    search: string;
}

export function PageDetails({ page, siteId, search }: PageDetailsProps): ReactElement {
    const update = useUpdatePage();
    const [draft, setDraft] = useState<Draft>(() => draftOf(page));

    useEffect(() => {
        setDraft(draftOf(page));
        update.reset();
    }, [page.id, page.updatedAt]);

    const edit = (patch: Partial<Draft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    const original = draftOf(page);
    const keys = Object.keys(original) as (keyof Draft)[];
    const dirty = keys.some((key) => original[key] !== draft[key]);

    const save = (): void => {
        const request: Parameters<typeof update.mutate>[0] = { id: page.id };
        for (const key of keys) {
            if (original[key] !== draft[key]) {
                request[key] = draft[key];
            }
        }
        update.mutate(request);
    };

    return (
        <div className="flex flex-col gap-3 p-3">
            {page.drift ? <DriftNotice page={page} /> : null}
            <Field
                label={copy.pages.detail.path}
                tooltip={copy.pages.detail.pathHint}
                error={fieldErrorOf(update.error, "path")}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        mono={true}
                        value={draft.path}
                        onChange={(event) => {
                            edit({ path: event.target.value });
                        }}
                    />
                )}
            </Field>
            <div className="grid grid-cols-2 gap-2">
                <Field label={copy.pages.detail.wpType} error={fieldErrorOf(update.error, "wpType")}>
                    {(control) => (
                        <Select
                            id={control.id}
                            value={draft.wpType}
                            options={wpTypeOptions}
                            invalid={control.invalid}
                            onValueChange={(next) => {
                                edit({ wpType: next });
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.pages.detail.status} error={fieldErrorOf(update.error, "status")}>
                    {(control) => (
                        <Select
                            id={control.id}
                            value={draft.status}
                            options={statusOptions}
                            invalid={control.invalid}
                            onValueChange={(next) => {
                                edit({ status: next });
                            }}
                        />
                    )}
                </Field>
            </div>
            <Field label={copy.pages.detail.title} error={fieldErrorOf(update.error, "title")}>
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={draft.title}
                        onChange={(event) => {
                            edit({ title: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.pages.detail.h1} error={fieldErrorOf(update.error, "h1")}>
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={draft.h1}
                        onChange={(event) => {
                            edit({ h1: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field
                label={copy.pages.detail.metaTitle}
                hint={copy.pages.detail.characters(draft.metaTitle.length)}
                error={fieldErrorOf(update.error, "metaTitle")}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={draft.metaTitle}
                        onChange={(event) => {
                            edit({ metaTitle: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field
                label={copy.pages.detail.metaDescription}
                hint={copy.pages.detail.characters(draft.metaDescription.length)}
                error={fieldErrorOf(update.error, "metaDescription")}
            >
                {(control) => (
                    <Textarea
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        rows={3}
                        value={draft.metaDescription}
                        onChange={(event) => {
                            edit({ metaDescription: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.pages.detail.canonicalUrl} error={fieldErrorOf(update.error, "canonical")}>
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        mono={true}
                        value={draft.canonical}
                        onChange={(event) => {
                            edit({ canonical: event.target.value });
                        }}
                    />
                )}
            </Field>
            {formErrorOf(update.error) === null ? null : (
                <p className="text-xs text-danger">{formErrorOf(update.error)}</p>
            )}
            <ConflictNotice thrown={update.error} siteId={siteId} search={search} />
            <div className="flex justify-end gap-2">
                <Button
                    variant="ghost"
                    disabled={!dirty}
                    onClick={() => {
                        setDraft(original);
                        update.reset();
                    }}
                >
                    {copy.pages.detail.revert}
                </Button>
                <Button variant="primary" disabled={!dirty} busy={update.isPending} onClick={save}>
                    {copy.pages.detail.save}
                </Button>
            </div>
        </div>
    );
}
