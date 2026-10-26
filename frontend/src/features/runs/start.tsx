import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { usePagePaths } from "../../data/hooks/pages.js";
import { useEstimateRun, useStartRun } from "../../data/hooks/runs.js";
import { useResolvedTemplate, useTemplates } from "../../data/hooks/templates.js";
import type { AddedPage, Estimate } from "../../data/types.js";
import { publishModes } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Button, CalculateIcon, Drawer, Field, PlayArrowIcon, Select } from "../../ui/index.js";
import { publishModeLabel, runKindLabel } from "./labels.js";
import {
    blockingFindings,
    capOf,
    kindDoes,
    startableKinds,
    startRefusal,
    startRequestOf,
    takesATemplate,
    templateAuto,
    tokenCapOf,
} from "./start-model.js";
import { StartCaps, StartPrice } from "./start-price.js";
import { RewriteSwitch, rewriteOf, writesBody } from "./start-rewrite.js";
import { scopedFields, StartScope, TypeScope } from "./start-scope.js";
import type { StartSeed } from "./start-seed.js";
import { seededFields, seededRewrite } from "./start-seed.js";
import { TargetTree } from "./start-tree.js";
import { kindGenerate } from "./statuses.js";

const drawerWidth = 688;
const templateAutoValue = "auto";
const listSize = 100;
const defaultCap = "5.00";
const defaultTokenCap = "0";

export interface StartRunDrawerProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    siteId: string;
    preselect?: readonly string[];
    initialKind?: string;
    seed?: StartSeed | null;
    onStarted: (runId: string) => void;
}

export function StartRunDrawer({
    open,
    onOpenChange,
    siteId,
    preselect,
    initialKind,
    seed = null,
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
    const [scoped, setScoped] = useState<readonly string[]>([]);
    const [rewrite, setRewrite] = useState<ReadonlySet<string>>(new Set());
    const [named, setNamed] = useState<ReadonlyMap<string, string>>(new Map());

    const priced = useEstimateRun();
    const start = useStartRun();
    const siteTemplates = useTemplates({ siteId }, { field: "name", desc: false }, listSize);
    const globalTemplates = useTemplates({ scope: "global" }, { field: "name", desc: false }, listSize);
    const first = [...selected][0] ?? null;
    const resolved = useResolvedTemplate(open && takesATemplate(kind) ? first : null, templateId);
    const fields = seededFields(seed, kind);
    const seededPages = seededRewrite(seed, kind);
    const offersRewrite =
        takesATemplate(kind) && (seededPages.length > 0 || (resolved.data !== undefined && writesBody(resolved.data.spec.recipe)));
    const rewritten = offersRewrite ? rewriteOf(rewrite, selected) : [];
    const unnamed = useMemo(() => [...selected].filter((id) => !named.has(id)), [selected, named]);
    const fetched = usePagePaths(unnamed, open && (offersRewrite || fields.length > 0));
    const pathOf = (id: string): string => named.get(id) ?? fetched.get(id) ?? id;

    useEffect(() => {
        if (!open) {
            return;
        }
        setSelected(new Set(seed?.pageIds ?? preselect ?? []));
        setPublishMode(publishModes[0]);
        setKind(seed?.kind ?? initialKind ?? kindGenerate);
        setTemplateId(templateAuto);
        setCap(defaultCap);
        setTokenCap(defaultTokenCap);
        setEstimate(null);
        setAdded([]);
        setScoped([]);
        setRewrite(new Set(seed?.rewrite ?? []));
        setNamed(new Map());
        priced.reset();
        start.reset();
    }, [open, siteId, seed]);

    if (!open) {
        return null;
    }

    const forget = (): void => {
        setEstimate(null);
        setAdded([]);
        setScoped([]);
        priced.reset();
    };

    const templateOptions: SelectOption<string>[] = [
        { value: templateAutoValue, label: copy.runs.start.templateAuto },
        ...flatten(siteTemplates.data?.pages).map((template) => ({ value: template.id, label: template.name })),
        ...flatten(globalTemplates.data?.pages).map((template) => ({ value: template.id, label: template.name })),
    ];

    const request = () =>
        startRequestOf({ siteId, pageIds: [...selected], publishMode, kind, templateId, cap, tokenCap, rewrite: rewritten, fields });

    const capValue = capOf(cap);
    const ready = selected.size > 0;
    const over = estimate !== null && capValue > 0 && estimate.usd > capValue;
    const blocked = blockingFindings(estimate);
    const refusal = startRefusal(start.error ?? priced.error);

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
                                    setScoped(scopedFields(answered));
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
                    onChange={(next, pages) => {
                        setSelected(next);
                        setNamed((held) => new Map([...held, ...pages.map((page): [string, string] => [page.id, page.path])]));
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
                                options={startableKinds.map((value) => ({ value, label: runKindLabel(value) }))}
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

                {fields.length === 0 ? null : <StartScope fields={fields} rewritten={rewritten.map(pathOf)} />}

                {offersRewrite && selected.size > 0 ? (
                    <RewriteSwitch
                        pages={[...selected].map((id) => ({ id, path: pathOf(id) }))}
                        rewrite={rewrite}
                        onChange={(next) => {
                            setRewrite(next);
                            forget();
                        }}
                    />
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

                {fields.length === 0 && scoped.length > 0 ? <TypeScope fields={scoped} /> : null}

                <StartPrice estimate={estimate} added={added} over={over} refusal={refusal.banner} />
            </div>
        </Drawer>
    );
}
