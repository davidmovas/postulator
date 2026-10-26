import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import type { Plan, PlanStep, PlanStepChoice } from "../../../data/types.js";
import { tokens as formatTokens, usd as formatUsd } from "../../../domain/format.js";
import {
    Banner,
    Button,
    cx,
    Input,
    OpenInNewIcon,
    PendingActionsIcon,
    Spinner,
    StatusBadge,
    Textarea,
    toneClasses,
} from "../../../ui/index.js";
import { editLabel, planStatusLabel, planStatusTone, stepStatusLabel, stepStatusTone } from "./labels.js";
import type { Drafts } from "./model.js";
import { choicesOf, costOf, dependentsOf, everyStep, stepsOf, ticked } from "./model.js";

const said = copy.plans;

const longEdits: ReadonlySet<string> = new Set(["notes", "reference", "styleSample"]);
const longValueRunes = 80;

export type PlanBusy = "all" | "chosen" | "decline" | null;

function estimateOf(step: PlanStep): string | null {
    const usd = step.estimateUsd ?? 0;
    const counted = step.estimateTokens ?? 0;
    if (usd === 0 && counted === 0) {
        return null;
    }
    return said.estimate(formatUsd(usd), formatTokens(counted));
}

interface MarkerProps {
    step: PlanStep;
    choosing: boolean;
    on: boolean;
    dependents: number;
    onTick: (on: boolean) => void;
}

function Marker({ step, choosing, on, dependents, onTick }: MarkerProps): ReactElement {
    if (choosing) {
        return (
            <label
                title={dependents === 0 ? undefined : said.tickDependents(dependents)}
                className="relative z-10 flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center"
            >
                <input
                    type="checkbox"
                    aria-label={said.tick(step.number)}
                    checked={on}
                    onChange={(event) => {
                        onTick(event.currentTarget.checked);
                    }}
                    className="peer sr-only"
                />
                <span
                    aria-hidden={true}
                    className={cx(
                        "flex h-5 w-5 items-center justify-center rounded-sm border font-mono text-2xs font-semibold transition-colors duration-100",
                        "peer-focus-visible:outline peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-accent",
                        on ? "border-accent bg-accent text-on-accent" : "border-edge bg-inset text-ink-faint",
                    )}
                >
                    {step.number}
                </span>
            </label>
        );
    }
    const tone = toneClasses[stepStatusTone(step.status)];
    return (
        <span
            aria-hidden={true}
            className={cx(
                "relative z-10 flex h-5 w-5 shrink-0 items-center justify-center rounded-sm border font-mono text-2xs font-semibold",
                tone.border,
                tone.soft,
                tone.ink,
            )}
        >
            {step.status === "running" ? <Spinner size={10} /> : step.number}
        </span>
    );
}

interface EditsProps {
    step: PlanStep;
    draft: Readonly<Record<string, string>>;
    onEdit: (name: string, value: string) => void;
}

function Edits({ step, draft, onEdit }: EditsProps): ReactElement {
    return (
        <div
            className="mt-1 flex flex-col gap-1.5 rounded-md border border-hairline bg-inset px-2.5 py-2"
            data-plan-edits={step.id}
        >
            <span className="text-2xs text-ink-faint">{said.editsHint}</span>
            {(step.edits ?? []).map((edit) => {
                const value = draft[edit.name] ?? edit.value;
                const long = longEdits.has(edit.name) || [...edit.value].length > longValueRunes;
                const label = editLabel(edit.name);
                return (
                    <label key={edit.name} className="flex flex-col gap-0.5">
                        <span className="text-2xs font-medium text-ink-soft">{label}</span>
                        {long ? (
                            <Textarea
                                rows={3}
                                value={value}
                                data-plan-edit={edit.name}
                                onChange={(event) => {
                                    onEdit(edit.name, event.target.value);
                                }}
                            />
                        ) : (
                            <Input
                                size="sm"
                                value={value}
                                data-plan-edit={edit.name}
                                onChange={(event) => {
                                    onEdit(edit.name, event.target.value);
                                }}
                            />
                        )}
                    </label>
                );
            })}
        </div>
    );
}

interface StepRowProps {
    step: PlanStep;
    steps: readonly PlanStep[];
    siteId: string | null;
    choosing: boolean;
    on: boolean;
    draft: Readonly<Record<string, string>>;
    onTick: (on: boolean) => void;
    onEdit: (name: string, value: string) => void;
}

function StepRow({ step, steps, siteId, choosing, on, draft, onTick, onEdit }: StepRowProps): ReactElement {
    const after = step.dependsOn ?? [];
    const estimate = estimateOf(step);
    const quiet = choosing ? !on : step.status === "declined" || step.status === "skipped";
    const tone = stepStatusTone(step.status);
    const edits = step.edits ?? [];
    const runId = step.runId ?? "";
    const error = step.error ?? "";

    return (
        <li
            data-plan-step={step.id}
            data-plan-step-status={step.status}
            data-plan-ticked={choosing ? on : undefined}
            className="relative flex gap-2.5 pb-3 last:pb-0 [&:last-child>span[data-rail]]:hidden"
        >
            <span
                data-rail={true}
                aria-hidden={true}
                className="absolute top-5 bottom-0 left-[9.5px] w-px bg-hairline"
            />
            <Marker
                step={step}
                choosing={choosing}
                on={on}
                dependents={dependentsOf(steps, step.id).size}
                onTick={onTick}
            />
            <div className="flex min-w-0 flex-1 flex-col gap-0.5 pt-px">
                <span className={cx("text-xs leading-snug", quiet ? "text-ink-faint" : "text-ink")}>
                    {step.summary}
                </span>
                <span className="text-2xs leading-snug text-ink-dim">{step.reason}</span>
                <span className="flex flex-wrap items-center gap-x-2 gap-y-0.5 text-2xs text-ink-faint">
                    {after.length === 0 ? null : <span>{said.after(after)}</span>}
                    {estimate === null ? null : <span className="font-mono">{estimate}</span>}
                    {choosing ? null : (
                        <span className={cx("font-semibold", toneClasses[tone].ink)}>
                            {stepStatusLabel(step.status)}
                        </span>
                    )}
                    {runId === "" || siteId === null || choosing ? null : (
                        <Link
                            to={`/s/${siteId}/runs/${runId}`}
                            className="inline-flex items-center gap-0.5 font-semibold text-accent hover:underline"
                        >
                            {said.openRun}
                            <OpenInNewIcon size={12} />
                        </Link>
                    )}
                </span>
                {error === "" || choosing ? null : (
                    <span
                        className={cx(
                            "text-2xs leading-snug",
                            step.status === "failed" ? "text-danger" : "text-ink-dim",
                        )}
                    >
                        {error}
                    </span>
                )}
                {choosing && on && edits.length > 0 ? <Edits step={step} draft={draft} onEdit={onEdit} /> : null}
            </div>
        </li>
    );
}

function Footer({ plan }: { plan: Plan }): ReactElement {
    const steps = stepsOf(plan);
    const approved = steps.filter((step) => step.status !== "declined");
    const done = approved.filter((step) => step.status === "done").length;
    return (
        <p className="text-2xs text-ink-faint">
            {plan.status === "declined"
                ? said.declinedBody
                : plan.status === "done"
                  ? said.finishedBody(done, approved.length)
                  : said.steps(approved.length)}
        </p>
    );
}

export interface PlanCardProps {
    plan: Plan;
    siteId: string | null;
    busy: PlanBusy;
    refusal: string | null;
    onConfirm: (choices: PlanStepChoice[], busy: Exclude<PlanBusy, null>) => void;
}

export function PlanCard({ plan, siteId, busy, refusal, onConfirm }: PlanCardProps): ReactElement {
    const steps = stepsOf(plan);
    const [ticks, setTicks] = useState<ReadonlySet<string>>(() => everyStep(plan));
    const [drafts, setDrafts] = useState<Drafts>(() => new Map());
    const choosing = plan.status === "proposed";
    const tone = planStatusTone(plan.status);
    const classes = toneClasses[tone];

    useEffect(() => {
        setTicks(everyStep(plan));
        setDrafts(new Map());
    }, [plan.id]);

    const cost = costOf(steps, ticks);
    const chosen = steps.filter((step) => ticks.has(step.id)).length;
    const every = everyStep(plan);

    const edit = (stepId: string, name: string, value: string): void => {
        setDrafts((held) => {
            const next = new Map(held);
            next.set(stepId, { ...(held.get(stepId) ?? {}), [name]: value });
            return next;
        });
    };

    return (
        <div
            role="group"
            aria-label={said.group(plan.title)}
            data-plan={plan.id}
            data-plan-status={plan.status}
            className={cx(
                "flex shrink-0 flex-col overflow-hidden rounded-lg border bg-panel",
                choosing ? classes.border : "border-hairline",
            )}
        >
            <div
                className={cx(
                    "flex items-center gap-2 border-b px-3 py-1.5",
                    choosing ? cx(classes.border, classes.soft) : "border-hairline",
                )}
            >
                <StatusBadge tone={tone} icon={PendingActionsIcon}>
                    {said.kicker}
                </StatusBadge>
                <span
                    className={cx(
                        "min-w-0 flex-1 truncate text-xs font-semibold",
                        choosing ? classes.ink : "text-ink-dim",
                    )}
                >
                    {planStatusLabel(plan.status)}
                </span>
                <span className="shrink-0 font-mono text-2xs text-ink-faint">{said.steps(steps.length)}</span>
            </div>
            <div className="flex flex-col gap-3 p-3">
                <span className="text-sm leading-snug font-semibold text-ink">{plan.title}</span>
                <ol className="flex flex-col">
                    {steps.map((step) => (
                        <StepRow
                            key={step.id}
                            step={step}
                            steps={steps}
                            siteId={siteId}
                            choosing={choosing}
                            on={ticks.has(step.id)}
                            draft={drafts.get(step.id) ?? {}}
                            onTick={(on) => {
                                setTicks((held) => ticked(steps, held, step.id, on));
                            }}
                            onEdit={(name, value) => {
                                edit(step.id, name, value);
                            }}
                        />
                    ))}
                </ol>
                {refusal === null ? null : <Banner tone="danger" title={refusal} />}
                {choosing ? (
                    <div className="flex flex-col gap-2 border-t border-hairline pt-2.5">
                        <span className="text-2xs text-ink-faint" data-plan-total={true}>
                            {chosen === 0
                                ? said.nothingChosen
                                : said.total(chosen, steps.length, formatUsd(cost.usd), cost.priced)}
                        </span>
                        <div className="flex flex-wrap items-center gap-2">
                            <Button
                                variant="primary"
                                data-plan-approve-all={true}
                                busy={busy === "all"}
                                disabled={busy !== null}
                                onClick={() => {
                                    setTicks(every);
                                    onConfirm(choicesOf(steps, every, drafts), "all");
                                }}
                            >
                                {busy === "all" ? said.approving : said.approveAll}
                            </Button>
                            <Button
                                variant="secondary"
                                data-plan-approve-chosen={true}
                                busy={busy === "chosen"}
                                disabled={busy !== null || chosen === 0 || chosen === steps.length}
                                title={chosen === steps.length ? said.approveChosenHint : undefined}
                                onClick={() => {
                                    onConfirm(choicesOf(steps, ticks, drafts), "chosen");
                                }}
                            >
                                {busy === "chosen" ? said.approving : said.approveChosen(chosen)}
                            </Button>
                            <Button
                                variant="ghost"
                                data-plan-decline={true}
                                busy={busy === "decline"}
                                disabled={busy !== null}
                                onClick={() => {
                                    onConfirm(choicesOf(steps, new Set(), drafts), "decline");
                                }}
                            >
                                {busy === "decline" ? said.declining : said.decline}
                            </Button>
                        </div>
                    </div>
                ) : (
                    <Footer plan={plan} />
                )}
            </div>
        </div>
    );
}
