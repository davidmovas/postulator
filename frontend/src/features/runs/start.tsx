import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useEstimateRun, useStartRun } from "../../data/hooks/runs.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { Estimate } from "../../data/types.js";
import { tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import { publishModes, runKinds } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import {
    Banner,
    Button,
    CalculateIcon,
    Drawer,
    Field,
    Input,
    PlayArrowIcon,
    SectionLabel,
    Select,
} from "../../ui/index.js";
import { StartTargets } from "./start-targets.js";
import { kindGenerate } from "./statuses.js";

const drawerWidth = 688;
const templateAuto = "";
const templateAutoValue = "auto";
const listSize = 100;
const defaultCap = "5.00";

function capOf(raw: string): number {
    const parsed = Number.parseFloat(raw);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
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
    const [selected, setSelected] = useState<readonly string[]>([]);
    const [publishMode, setPublishMode] = useState<string>(publishModes[0]);
    const [kind, setKind] = useState<string>(initialKind ?? kindGenerate);
    const [templateId, setTemplateId] = useState<string>(templateAuto);
    const [cap, setCap] = useState(defaultCap);
    const [estimate, setEstimate] = useState<Estimate | null>(null);

    const priced = useEstimateRun();
    const start = useStartRun();
    const siteTemplates = useTemplates({ siteId }, { field: "name", desc: false }, listSize);
    const globalTemplates = useTemplates({ scope: "global" }, { field: "name", desc: false }, listSize);

    useEffect(() => {
        if (!open) {
            return;
        }
        setSelected(preselect ?? []);
        setPublishMode(publishModes[0]);
        setKind(initialKind ?? kindGenerate);
        setTemplateId(templateAuto);
        setCap(defaultCap);
        setEstimate(null);
        priced.reset();
        start.reset();
    }, [open, siteId]);

    if (!open) {
        return null;
    }

    const forget = (): void => {
        setEstimate(null);
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
            budget: { maxUsd: capOf(cap), maxTokens: 0 },
        };
        if (templateId !== templateAuto) {
            built.templateId = templateId;
        }
        return built;
    };

    const capValue = capOf(cap);
    const ready = selected.length > 0;
    const over = estimate !== null && capValue > 0 && estimate.usd > capValue;
    const thrown = start.error ?? priced.error;
    const reaction = thrown === null || thrown === undefined ? null : react(thrown);
    const problem =
        reaction === null || reaction.kind === "silent" || reaction.kind === "unlock" ? null : reaction.message;

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
                                },
                            });
                        }}
                    >
                        {copy.runs.start.estimate}
                    </Button>
                    <Button
                        variant="primary"
                        icon={PlayArrowIcon}
                        disabled={estimate === null}
                        busy={start.isPending}
                        title={estimate === null ? copy.runs.start.estimateFirst : undefined}
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
                <StartTargets
                    siteId={siteId}
                    selected={selected}
                    onToggle={(pageId) => {
                        setSelected((held) =>
                            held.includes(pageId) ? held.filter((id) => id !== pageId) : [...held, pageId],
                        );
                        forget();
                    }}
                />

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

                <Field
                    label={copy.runs.start.cap}
                    tooltip={copy.runs.start.capHint}
                    hint={capValue === 0 ? copy.runs.start.capZero : undefined}
                >
                    {(control) => (
                        <Input
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            mono={true}
                            inputMode="decimal"
                            value={cap}
                            onChange={(event) => {
                                setCap(event.target.value);
                                forget();
                            }}
                        />
                    )}
                </Field>

                {estimate === null ? null : (
                    <div className="flex items-baseline justify-between gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2">
                        <SectionLabel>{copy.runs.start.estimate}</SectionLabel>
                        <span className="font-mono text-sm text-ink">
                            {copy.runs.start.estimated(formatUsd(estimate.usd), formatTokens(estimate.tokens))}
                        </span>
                    </div>
                )}

                {over ? <Banner tone="warn" title={copy.runs.start.overCap} /> : null}
                {problem === null ? null : <Banner tone="danger" title={problem} />}
            </div>
        </Drawer>
    );
}

export const StartRunDialog = StartRunDrawer;
