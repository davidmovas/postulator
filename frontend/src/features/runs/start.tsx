import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useEstimateRun, useStartRun } from "../../data/hooks/runs.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { AddedPage, Estimate } from "../../data/types.js";
import { publishModes, runKinds, runKindsWithTheirOwnRecipe } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, CalculateIcon, Drawer, Field, PlayArrowIcon, Select } from "../../ui/index.js";
import { kindLabel, publishModeLabel } from "./labels.js";
import { StartCaps, StartPrice } from "./start-price.js";
import { TargetTree } from "./start-tree.js";
import { kindGenerate, kindRevert } from "./statuses.js";

const drawerWidth = 688;
const templateAuto = "";
const templateAutoValue = "auto";
const listSize = 100;
const defaultCap = "5.00";
const defaultTokenCap = "0";

function capOf(raw: string): number {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

export function tokenCapOf(raw: string): number {
    const parsed = Number.parseInt(raw, 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

export function kindDoes(kind: string): string {
    const said: Record<string, string> = copy.runs.start.kindDoes;
    return said[kind] ?? "";
}

export function takesATemplate(kind: string): boolean {
    return !(runKindsWithTheirOwnRecipe as readonly string[]).includes(kind);
}

export interface GradedFinding {
    severity: string;
    code?: string;
    message?: string;
    pageId?: string;
    path?: string;
}

export interface GradedFindings {
    findings?: readonly GradedFinding[] | null;
}

export function blockingFindings(estimate: GradedFindings | null): number {
    if (estimate === null) {
        return 0;
    }
    return (estimate.findings ?? []).filter((finding) => finding.severity === "error").length;
}

export const startableKinds: readonly string[] = runKinds.filter((kind) => kind !== kindRevert);

export interface StartRefusal {
    targets: string | null;
    banner: string | null;
}

export function startRefusal(thrown: unknown): StartRefusal {
    if (thrown === null || thrown === undefined) {
        return { targets: null, banner: null };
    }
    const reaction = react(thrown);
    if (reaction.kind === "silent" || reaction.kind === "unlock") {
        return { targets: null, banner: null };
    }
    if (reaction.kind === "field" && reaction.field === "pageIds") {
        return { targets: reaction.message, banner: null };
    }
    return { targets: null, banner: reaction.message };
}


export interface StartRunDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    preselect?: readonly string[];
    initialKind?: string;
    onStarted: (runId: string) => void;
}

export function StartRunDrawer({
    open,
    onOpenChange,
    siteId,
    preselect,
    initialKind,
    onStarted,
}: StartRunDrawerProps): ReactElement | null {
    const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
    const [publishMode, setPublishMode] = useState<string>(publishModes[0]);
    const [kind, setKind] = useState<string>(initialKind ?? kindGenerate);
    const [templateId, setTemplateId] = useState<string>(templateAuto);
    const [cap, setCap] = useState(defaultCap);
    const [tokenCap, setTokenCap] = useState(defaultTokenCap);
    const [estimate, setEstimate] = useState<Estimate | null>(null);
    const [added, setAdded] = useState<readonly AddedPage[]>([]);

    const priced = useEstimateRun();
    const start = useStartRun();
    const siteTemplates = useTemplates({ siteId }, { field: "name", desc: false }, listSize);
    const globalTemplates = useTemplates({ scope: "global" }, { field: "name", desc: false }, listSize);

    useEffect(() => {
        if (!open) {
            return;
        }
        setSelected(new Set(preselect ?? []));
        setPublishMode(publishModes[0]);
        setKind(initialKind ?? kindGenerate);
        setTemplateId(templateAuto);
        setCap(defaultCap);
        setTokenCap(defaultTokenCap);
        setEstimate(null);
        setAdded([]);
        priced.reset();
        start.reset();
    }, [open, siteId]);

    if (!open) {
        return null;
    }

    const forget = (): void => {
        setEstimate(null);
        setAdded([]);
        priced.reset();
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
            budget: { maxUsd: capOf(cap), maxTokens: tokenCapOf(tokenCap) },
        };
        if (templateId !== templateAuto) {
            built.templateId = templateId;
        }
        return built;
    };

    const capValue = capOf(cap);
    const ready = selected.size > 0;
    const over = estimate !== null && capValue > 0 && estimate.usd > capValue;
    const blocked = blockingFindings(estimate);
    const thrown = start.error ?? priced.error;
    const refusal = startRefusal(thrown);

    return (
        <Drawer
            open={true}
            onOpenChange={onOpenChange}
            title={copy.runs.start.title}
            closeLabel={copy.runs.start.cancel}
            width={drawerWidth}
            footer={
                <>
                    <Button
                        variant="ghost"
                        onClick={() => {
                            onOpenChange(false);
                        }}
                    >
                        {copy.runs.start.cancel}
                    </Button>
                    <Button
                        data-run-estimate={true}
                        icon={CalculateIcon}
                        disabled={!ready}
                        busy={priced.isPending}
                        title={ready ? undefined : copy.runs.start.noPages}
                        onClick={() => {
                            priced.mutate(request(), {
                                onSuccess: (answered) => {
                                    setEstimate(answered.estimate);
                                    setAdded(answered.added ?? []);
                                },
                            });
                        }}
                    >
                        {copy.runs.start.estimate}
                    </Button>
                    <Button
                        variant="primary"
                        icon={PlayArrowIcon}
                        disabled={estimate === null || blocked > 0}
                        busy={start.isPending}
                        title={
                            estimate === null
                                ? copy.runs.start.estimateFirst
                                : blocked > 0
                                  ? copy.runs.start.blocked
                                  : undefined
                        }
                        onClick={() => {
                            start.mutate(request(), {
                                onSuccess: (answered) => {
                                    onStarted(answered.runId);
                                },
                            });
                        }}
                    >
                        {copy.runs.start.confirm}
                    </Button>
                </>
            }
        >
            <div className="flex flex-col gap-3 p-3">
                <TargetTree
                    siteId={siteId}
                    selected={selected}
                    problem={refusal.targets}
                    onChange={(next) => {
                        setSelected(next);
                        forget();
                    }}
                />

                <div className="grid grid-cols-2 gap-2">
                    <Field label={copy.runs.start.publishMode}>
                        {(control) => (
                            <Select
                                id={control.id}
                                value={publishMode}
                                options={publishModes.map((mode) => ({ value: mode, label: publishModeLabel(mode) }))}
                                onValueChange={(next) => {
                                    setPublishMode(next);
                                    forget();
                                }}
                            />
                        )}
                    </Field>
                    <Field label={copy.runs.start.kind} hint={kindDoes(kind)}>
                        {(control) => (
                            <Select
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                value={kind}
                                options={startableKinds.map((value) => ({ value, label: kindLabel(value) }))}
                                onValueChange={(next) => {
                                    setKind(next);
                                    forget();
                                }}
                            />
                        )}
                    </Field>
                </div>

                {takesATemplate(kind) ? (
                    <Field label={copy.runs.start.template} tooltip={copy.runs.start.templateHint}>
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
                ) : null}

                <StartCaps
                    cap={cap}
                    tokenCap={tokenCap}
                    capValue={capValue}
                    tokenCapValue={tokenCapOf(tokenCap)}
                    onCap={(next) => {
                        setCap(next);
                        forget();
                    }}
                    onTokenCap={(next) => {
                        setTokenCap(next);
                        forget();
                    }}
                />

                <StartPrice estimate={estimate} added={added} over={over} refusal={refusal.banner} />
            </div>
        </Drawer>
    );
}

