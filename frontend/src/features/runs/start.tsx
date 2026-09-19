import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { usePages } from "../../data/hooks/pages.js";
import { useEstimateRun, useStartRun } from "../../data/hooks/runs.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { Estimate, PageFilter } from "../../data/types.js";
import { tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import { publishModes, runKinds } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import {
    Banner,
    Button,
    Checkbox,
    Dialog,
    Field,
    Input,
    PlayArrowIcon,
    SectionLabel,
    Select,
    Skeleton,
} from "../../ui/index.js";
import { kindGenerate } from "./statuses.js";

const templateAuto = "";
const templateAutoValue = "auto";
const pageSize = 100;
const defaultCap = "5.00";

function capOf(raw: string): number {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

export interface StartRunDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    preselect?: readonly string[];
    onStarted: (runId: string) => void;
}

export function StartRunDialog({
    open,
    onOpenChange,
    siteId,
    preselect,
    onStarted,
}: StartRunDialogProps): ReactElement {
    const [prefix, setPrefix] = useState("");
    const [applied, setApplied] = useState("");
    const [selected, setSelected] = useState<readonly string[]>([]);
    const [publishMode, setPublishMode] = useState<string>(publishModes[0]);
    const [kind, setKind] = useState<string>(kindGenerate);
    const [templateId, setTemplateId] = useState<string>(templateAuto);
    const [cap, setCap] = useState(defaultCap);
    const [estimate, setEstimate] = useState<Estimate | null>(null);

    const priced = useEstimateRun();
    const start = useStartRun();

    const filter = useMemo<PageFilter>(
        () => (applied === "" ? { siteId } : { siteId, pathPrefix: applied }),
        [siteId, applied],
    );
    const listed = usePages(filter, { field: "path", desc: false }, pageSize);
    const pages = useMemo(() => flatten(listed.data?.pages), [listed.data]);

    const siteTemplates = useTemplates({ siteId }, { field: "name", desc: false }, pageSize);
    const globalTemplates = useTemplates({ scope: "global" }, { field: "name", desc: false }, pageSize);

    useEffect(() => {
        if (!open) {
            return;
        }
        setPrefix("");
        setApplied("");
        setSelected(preselect ?? []);
        setPublishMode(publishModes[0]);
        setKind(kindGenerate);
        setTemplateId(templateAuto);
        setCap(defaultCap);
        setEstimate(null);
        priced.reset();
        start.reset();
    }, [open, siteId]);

    const forget = (): void => {
        setEstimate(null);
        priced.reset();
    };

    const toggle = (pageId: string): void => {
        setSelected((held) => (held.includes(pageId) ? held.filter((id) => id !== pageId) : [...held, pageId]));
        forget();
    };

    const templateOptions: SelectOption<string>[] = [
        { value: templateAutoValue, label: copy.runs.start.templateAuto },
        ...flatten(siteTemplates.data?.pages).map((template) => ({ value: template.id, label: template.name })),
        ...flatten(globalTemplates.data?.pages).map((template) => ({ value: template.id, label: template.name })),
    ];

    const request = (): Parameters<typeof start.mutate>[0] => {
        const built: Parameters<typeof start.mutate>[0] = {
            siteId,
            pageIds: [...selected],
            publishMode,
            kind,
            budget: { maxUsd: capOf(cap), maxTokens: 0 },
        };
        if (templateId !== templateAuto) {
            built.templateId = templateId;
        }
        return built;
    };

    const ready = selected.length > 0;
    const busy = priced.isPending || start.isPending;
    const capValue = capOf(cap);
    const over = estimate !== null && capValue > 0 && estimate.usd > capValue;

    const confirm = (): void => {
        if (!ready || busy) {
            return;
        }
        if (estimate === null) {
            priced.mutate(request(), {
                onSuccess: (answered) => {
                    setEstimate(answered.estimate);
                },
            });
            return;
        }
        start.mutate(request(), {
            onSuccess: (answered) => {
                onStarted(answered.runId);
            },
        });
    };

    const failure = start.error ?? priced.error;
    const reaction = failure === null || failure === undefined ? null : react(failure);
    const problem =
        reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;

    return (
        <Dialog
            open={open}
            onOpenChange={onOpenChange}
            title={copy.runs.start.title}
            description={copy.runs.start.body}
            icon={PlayArrowIcon}
            confirmLabel={estimate === null ? copy.runs.start.estimate : copy.runs.start.confirm}
            onConfirm={confirm}
            cancelLabel={copy.runs.start.cancel}
            busy={busy}
        >
            <div className="flex flex-col gap-3 pt-1">
                <div className="flex flex-col gap-1">
                    <div className="flex items-center justify-between gap-2">
                        <SectionLabel>{copy.runs.start.pages}</SectionLabel>
                        <span className="font-mono text-2xs text-ink-faint">
                            {copy.runs.start.selected(selected.length)}
                        </span>
                    </div>
                    <Input
                        mono={true}
                        value={prefix}
                        placeholder="/"
                        aria-label={copy.runs.start.search}
                        onChange={(event) => {
                            setPrefix(event.target.value);
                        }}
                        onBlur={() => {
                            setApplied(prefix.trim());
                        }}
                        onKeyDown={(event) => {
                            if (event.key === "Enter") {
                                event.preventDefault();
                                setApplied(prefix.trim());
                            }
                        }}
                    />
                    <div className="flex max-h-44 min-h-24 flex-col gap-0.5 overflow-auto rounded-md border border-hairline bg-inset p-1.5">
                        {listed.isPending ? (
                            <Skeleton height={14} />
                        ) : pages.length === 0 ? (
                            <p className="p-1 text-xs text-ink-dim">{copy.empty.pages}</p>
                        ) : (
                            pages.map((page) => (
                                <Checkbox
                                    key={page.id}
                                    checked={selected.includes(page.id)}
                                    label={page.path}
                                    onChange={() => {
                                        toggle(page.id);
                                    }}
                                />
                            ))
                        )}
                        {listed.hasNextPage ? (
                            <Button
                                size="sm"
                                variant="ghost"
                                busy={listed.isFetchingNextPage}
                                onClick={() => {
                                    void listed.fetchNextPage();
                                }}
                            >
                                {copy.runs.loadMore}
                            </Button>
                        ) : null}
                    </div>
                    <p className="text-2xs text-ink-faint">{copy.runs.start.pagesHint}</p>
                </div>

                <div className="grid grid-cols-2 gap-2">
                    <Field label={copy.runs.start.publishMode}>
                        {(control) => (
                            <Select
                                id={control.id}
                                value={publishMode}
                                options={publishModes.map((mode) => ({ value: mode, label: mode }))}
                                onValueChange={(next) => {
                                    setPublishMode(next);
                                    forget();
                                }}
                            />
                        )}
                    </Field>
                    <Field label={copy.runs.start.kind}>
                        {(control) => (
                            <Select
                                id={control.id}
                                value={kind}
                                options={runKinds.map((value) => ({ value, label: value }))}
                                onValueChange={(next) => {
                                    setKind(next);
                                    forget();
                                }}
                            />
                        )}
                    </Field>
                </div>

                <Field label={copy.runs.start.template} hint={copy.runs.start.templateHint}>
                    {(control) => (
                        <Select
                            id={control.id}
                            value={templateId === templateAuto ? templateAutoValue : templateId}
                            options={templateOptions}
                            onValueChange={(next) => {
                                setTemplateId(next === templateAutoValue ? templateAuto : next);
                                forget();
                            }}
                        />
                    )}
                </Field>

                <Field
                    label={copy.runs.start.cap}
                    hint={capValue === 0 ? copy.runs.start.capZero : copy.runs.start.capHint}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            mono={true}
                            inputMode="decimal"
                            value={cap}
                            onChange={(event) => {
                                setCap(event.target.value);
                            }}
                        />
                    )}
                </Field>

                {estimate === null ? (
                    <p className="text-xs text-ink-dim">
                        {ready ? copy.runs.start.estimateFirst : copy.runs.start.noPages}
                    </p>
                ) : (
                    <div className="flex flex-col gap-1 rounded-md border border-hairline bg-inset px-2.5 py-2">
                        <div className="flex items-baseline justify-between gap-2">
                            <span className="text-2xs tracking-label text-ink-faint uppercase">
                                {copy.runs.start.estimate}
                            </span>
                            <span className="font-mono text-sm text-ink">
                                {copy.runs.start.estimated(formatUsd(estimate.usd), formatTokens(estimate.tokens))}
                            </span>
                        </div>
                        <p className="text-2xs text-ink-faint">{copy.runs.start.estimateHint}</p>
                    </div>
                )}

                {over ? <Banner tone="warn" title={copy.runs.start.overCap} /> : null}
                {problem === null ? null : <Banner tone="danger" title={problem} />}
            </div>
        </Dialog>
    );
}
