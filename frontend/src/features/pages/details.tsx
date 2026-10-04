import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, validationErrorOf } from "../../data/errors.js";
import { useUpdatePage } from "../../data/hooks/pages.js";
import type { Keyword, Page } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { keywordList, sameKeywords } from "../../domain/keywords.js";
import { pageStatuses, pageWpTypes } from "../../generated/vocab.js";
import { pageStatusLabel } from "./labels.js";
import type { SelectOption } from "../../ui/index.js";
import { Banner, Button, Field, Input, KeywordInput, Select, SyncProblemIcon, Textarea } from "../../ui/index.js";
import { ConflictNotice } from "./conflict-notice.js";
import { ProductPanel } from "./product-panel.js";

const wpTypeOptions: readonly SelectOption<string>[] = pageWpTypes.map((value) => ({ value, label: value }));
const statusOptions: readonly SelectOption<string>[] = pageStatuses.map((value) => ({
    value,
    label: pageStatusLabel(value),
}));

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
    const [keywords, setKeywords] = useState<Keyword[]>(() => keywordList(page.keywords));

    useEffect(() => {
        setDraft(draftOf(page));
        setKeywords(keywordList(page.keywords));
        update.reset();
    }, [page.id, page.updatedAt]);

    const edit = (patch: Partial<Draft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    const original = draftOf(page);
    const storedKeywords = keywordList(page.keywords);
    const keys = Object.keys(original) as (keyof Draft)[];
    const keywordsChanged = !sameKeywords(storedKeywords, keywords);
    const dirty = keywordsChanged || keys.some((key) => original[key] !== draft[key]);

    const save = (): void => {
        const request: Parameters<typeof update.mutate>[0] = { id: page.id };
        for (const key of keys) {
            if (original[key] !== draft[key]) {
                request[key] = draft[key];
            }
        }
        if (keywordsChanged) {
            request.keywords = keywords;
        }
        update.mutate(request);
    };

    return (
        <div className="flex flex-col gap-3 p-3">
            {page.drift ? <DriftNotice page={page} /> : null}
            {page.wpType === "product" ? <ProductPanel page={page} /> : null}
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
                label={copy.pages.detail.keywords}
                hint={keywords.length > 0 ? copy.pages.detail.keywordsOwn : copy.pages.detail.keywordsInherited}
                error={fieldErrorOf(update.error, "keywords")}
            >
                {(control) => (
                    <KeywordInput
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        values={keywords}
                        removeLabel={copy.pages.detail.removeKeyword}
                        phraseLabel={copy.pages.detail.keyword}
                        volumeLabel={copy.pages.detail.volume}
                        onChange={setKeywords}
                    />
                )}
            </Field>
            {(page.notes ?? []).length === 0 ? null : (
                <section aria-label={copy.pages.detail.notes} className="flex flex-col gap-1">
                    <span className="text-xs font-medium text-ink-dim">{copy.pages.detail.notes}</span>
                    <dl className="flex flex-col gap-1 rounded-md border border-hairline bg-inset px-2.5 py-2 text-xs">
                        {(page.notes ?? []).map((note) => (
                            <div key={note.label} className="flex gap-2">
                                <dt className="shrink-0 text-ink-faint">{note.label}</dt>
                                <dd className="min-w-0 text-ink-soft">{note.text}</dd>
                            </div>
                        ))}
                    </dl>
                    <p className="text-2xs text-ink-faint">{copy.pages.detail.notesHint}</p>
                </section>
            )}
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
            {validationErrorOf(update.error) === null ? null : (
                <p className="text-xs text-danger">{validationErrorOf(update.error)}</p>
            )}
            <ConflictNotice thrown={update.error} siteId={siteId} search={search} />
            <div className="flex justify-end gap-2">
                <Button
                    variant="ghost"
                    disabled={!dirty}
                    onClick={() => {
                        setDraft(original);
                        setKeywords(storedKeywords);
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
