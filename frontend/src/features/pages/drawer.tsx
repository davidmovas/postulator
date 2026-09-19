import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { failure, react } from "../../data/errors.js";
import { useDeletePage, usePage, useUpdatePage } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { pageStatuses, pageWpTypes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import {
    Banner,
    Button,
    DeleteIcon,
    Dialog,
    Drawer,
    Field,
    Input,
    Panel,
    PanelHeader,
    Select,
    SkeletonRows,
    SmartToyIcon,
    StatusBadge,
    SyncProblemIcon,
    Textarea,
} from "../../ui/index.js";
import { askAgent } from "../agent/dock-state.js";
import { ConflictNotice } from "./conflict-notice.js";
import type { EntityIndex } from "./entities.js";
import { statusTone } from "./labels.js";
import { PageLinks } from "./links.js";
import { PageMapping } from "./mapping.js";
import { PageReportPanel } from "./report.js";

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

interface DriftNoticeProps {
    page: Page;
}

function DriftNotice({ page }: DriftNoticeProps): ReactElement {
    return (
        <Banner
            tone="warn"
            icon={SyncProblemIcon}
            title={copy.pages.drift.title}
            body={
                <div className="flex flex-col gap-1">
                    <p>{copy.pages.drift.body}</p>
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

interface MetaFormProps {
    page: Page;
    siteId: string;
    search: string;
}

function MetaForm({ page, siteId, search }: MetaFormProps): ReactElement {
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
    const dirty = (Object.keys(original) as (keyof Draft)[]).some((key) => original[key] !== draft[key]);

    const save = (): void => {
        const request: Parameters<typeof update.mutate>[0] = { id: page.id };
        if (draft.path !== original.path) {
            request.path = draft.path;
        }
        if (draft.title !== original.title) {
            request.title = draft.title;
        }
        if (draft.h1 !== original.h1) {
            request.h1 = draft.h1;
        }
        if (draft.metaTitle !== original.metaTitle) {
            request.metaTitle = draft.metaTitle;
        }
        if (draft.metaDescription !== original.metaDescription) {
            request.metaDescription = draft.metaDescription;
        }
        if (draft.canonical !== original.canonical) {
            request.canonical = draft.canonical;
        }
        if (draft.wpType !== original.wpType) {
            request.wpType = draft.wpType;
        }
        if (draft.status !== original.status) {
            request.status = draft.status;
        }
        update.mutate(request);
    };

    return (
        <Panel>
            <PanelHeader title={copy.pages.detail.meta}>
                <div className="flex shrink-0 gap-1">
                    <Button
                        size="sm"
                        variant="ghost"
                        disabled={!dirty}
                        onClick={() => {
                            setDraft(original);
                            update.reset();
                        }}
                    >
                        {copy.pages.detail.revert}
                    </Button>
                    <Button
                        size="sm"
                        variant="primary"
                        disabled={!dirty}
                        busy={update.isPending}
                        onClick={save}
                    >
                        {copy.pages.detail.save}
                    </Button>
                </div>
            </PanelHeader>
            <div className="flex flex-col gap-2.5 p-3">
                <Field
                    label={copy.pages.detail.path}
                    hint={copy.pages.detail.pathHint}
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
                <Field
                    label={copy.pages.detail.canonicalUrl}
                    error={fieldErrorOf(update.error, "canonical")}
                >
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
            </div>
        </Panel>
    );
}

export interface PageDrawerProps {
    pageId: string;
    siteId: string;
    index: EntityIndex;
    search: string;
    onClose: () => void;
}

export function PageDrawer({ pageId, siteId, index, search, onClose }: PageDrawerProps): ReactElement {
    const detail = usePage(pageId);
    const remove = useDeletePage();
    const [confirming, setConfirming] = useState(false);
    const page = detail.data?.page;

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={page?.path ?? copy.app.loading}
            closeLabel={copy.pages.detail.close}
            width={560}
            header={
                page === undefined ? null : (
                    <div className="flex shrink-0 items-center gap-1.5">
                        <StatusBadge tone={statusTone(page.status)}>{page.status}</StatusBadge>
                        {page.drift ? (
                            <StatusBadge tone="warn" icon={SyncProblemIcon}>
                                {copy.pages.drift.badge}
                            </StatusBadge>
                        ) : null}
                    </div>
                )
            }
            footer={
                page === undefined ? null : (
                    <>
                        <Button
                            className="mr-auto"
                            variant="danger"
                            icon={DeleteIcon}
                            onClick={() => {
                                setConfirming(true);
                            }}
                        >
                            {copy.pages.detail.deletePage}
                        </Button>
                        <Button
                            variant="ghost"
                            icon={SmartToyIcon}
                            onClick={() => {
                                askAgent(copy.agent.ask.page(page.path, page.id));
                            }}
                        >
                            {copy.agent.askAbout}
                        </Button>
                        <Button onClick={onClose}>{copy.pages.detail.close}</Button>
                    </>
                )
            }
        >
            {detail.isPending ? (
                <div className="p-3">
                    <SkeletonRows rows={8} label={copy.app.loading} />
                </div>
            ) : page === undefined ? (
                <div className="p-3">
                    <p className="text-sm text-ink">
                        {failure(detail.error).code === "NOT_FOUND"
                            ? copy.pages.detail.notFound
                            : failure(detail.error).message}
                    </p>
                </div>
            ) : (
                <div className="flex flex-col gap-3 p-3">
                    <div className="flex flex-col gap-1">
                        <h3 className="text-lg font-semibold text-ink">
                            {page.title === "" ? copy.pages.untitled : page.title}
                        </h3>
                        <p className="font-mono text-xs text-ink-faint select-all">{page.path}</p>
                    </div>
                    {page.drift ? <DriftNotice page={page} /> : null}
                    <PageMapping page={page} siteId={siteId} index={index} search={search} />
                    <MetaForm page={page} siteId={siteId} search={search} />
                    <PageLinks links={detail.data?.links ?? []} siteId={siteId} search={search} />
                    <PageReportPanel pageId={page.id} siteId={siteId} />
                    <Dialog
                        open={confirming}
                        onOpenChange={setConfirming}
                        title={copy.pages.detail.deleteTitle}
                        description={copy.pages.detail.deleteBody}
                        confirmLabel={copy.pages.detail.deleteConfirm}
                        cancelLabel={copy.pages.detail.cancel}
                        destructive={true}
                        icon={DeleteIcon}
                        busy={remove.isPending}
                        onConfirm={() => {
                            remove.mutate(
                                { id: page.id },
                                {
                                    onSuccess: () => {
                                        setConfirming(false);
                                        onClose();
                                    },
                                },
                            );
                        }}
                    />
                </div>
            )}
        </Drawer>
    );
}
