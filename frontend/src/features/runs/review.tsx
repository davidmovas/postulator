import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useArtifacts, useResumeRun, useRetryStep } from "../../data/hooks/runs.js";
import { useSetting } from "../../data/hooks/settings.js";
import type { Page, RunItem } from "../../data/types.js";
import { duration } from "../../domain/format.js";
import type { ArtifactKind } from "../../generated/vocab.js";
import type { TabItem } from "../../ui/index.js";
import {
    Banner,
    Button,
    cx,
    Drawer,
    Kbd,
    OpenInNewIcon,
    RestartAltIcon,
    SkeletonRows,
    SmartToyIcon,
    StatusBadge,
    TabPanel,
    Tabs,
} from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { countdown, dueMs, remainingMs, retryState, waitingUntil } from "./authority.js";
import type { ItemView } from "./authority.js";
import {
    artifactLabel,
    orderedKinds,
    pauseReasonText,
    retryBlockedText,
    statusIcon,
    statusLabel,
    statusTone,
} from "./labels.js";
import type { RetryNotice, StepEntry } from "./log-view.js";
import { ArtifactPane } from "./panes/index.js";
import { driftRefusal } from "./refusal.js";
import { neighbourOf, positionOf } from "./review-nav.js";
import { DriftRefused, ReviewEmpty, ReviewMissing } from "./review-states.js";
import { artifactBodyHtml, retentionDaysKey } from "./statuses.js";
import { stepText } from "./step-cell.js";

const drawerWidth = 688;

function Timeline({ entries }: { entries: readonly StepEntry[] }): ReactElement | null {
    if (entries.length === 0) {
        return null;
    }
    return (
        <div className="flex flex-wrap items-center gap-1 border-b border-hairline px-3 py-1.5">
            {entries.map((entry, position) => (
                <span
                    key={`${entry.step}:${String(position)}`}
                    title={entry.message ?? undefined}
                    className={cx(
                        "inline-flex items-center gap-1 rounded-sm px-1.5 py-0.5 font-mono text-2xs",
                        entry.code !== null
                            ? "bg-danger-soft text-danger"
                            : entry.finishedAt === null
                              ? "bg-info-soft text-info"
                              : "bg-inset text-ink-dim",
                    )}
                >
                    {entry.step}
                    {entry.durationMs === null ? null : (
                        <span className="text-ink-faint">{duration(entry.durationMs)}</span>
                    )}
                    {entry.attempts > 1 ? (
                        <span className="text-warn">{copy.runs.events.attempt(entry.attempts)}</span>
                    ) : null}
                </span>
            ))}
        </div>
    );
}

function retentionOf(held: unknown): number | null {
    return typeof held === "number" && Number.isFinite(held) ? held : null;
}

function typing(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    return target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
}

export interface ReviewDrawerProps {
    view: ItemView | null;
    page: Page | undefined;
    steps: readonly string[];
    timeline: readonly StepEntry[];
    retry: RetryNotice | undefined;
    path: string;
    now: number;
    missing: boolean;
    narrowed: boolean;
    siblings: readonly RunItem[];
    onClearFilter: () => void;
    onMove: (itemId: string) => void;
    onOpenPage: (pageId: string) => void;
    onClose: () => void;
    onRerun: (pageId: string) => void;
}

export function ReviewDrawer({
    view,
    page,
    steps,
    timeline,
    retry,
    path,
    now,
    missing,
    narrowed,
    siblings,
    onClearFilter,
    onMove,
    onOpenPage,
    onClose,
    onRerun,
}: ReviewDrawerProps): ReactElement {
    const item: RunItem | null = view === null ? null : view.item;
    const itemId = item === null ? "" : item.id;
    const listed = useArtifacts(itemId === "" ? null : itemId);
    const retention = useSetting(retentionDaysKey);
    const retryStep = useRetryStep();
    const resume = useResumeRun();
    const [active, setActive] = useState<ArtifactKind | null>(null);

    const kinds = useMemo(
        () => orderedKinds((listed.data?.artifacts ?? []).map((artifact) => artifact.kind)),
        [listed.data],
    );

    useEffect(() => {
        setActive(null);
    }, [itemId]);

    useEffect(() => {
        if (active === null && kinds.length > 0) {
            setActive(kinds.includes(artifactBodyHtml) ? artifactBodyHtml : kinds[kinds.length - 1]);
        }
    }, [active, kinds]);

    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent): void => {
            if (event.ctrlKey || event.metaKey || event.altKey || typing(event.target)) {
                return;
            }
            const pressed = event.key.toLowerCase();
            if (pressed !== "j" && pressed !== "k") {
                return;
            }
            const next = neighbourOf(siblings, itemId, pressed === "j" ? 1 : -1);
            if (next !== null) {
                event.preventDefault();
                onMove(next);
            }
        };
        window.addEventListener("keydown", onKeyDown);
        return () => {
            window.removeEventListener("keydown", onKeyDown);
        };
    }, [siblings, itemId, onMove]);

    const retentionDays = retentionOf(retention.data?.value);
    const state = item === null ? null : retryState(item);
    const wake = item === null ? null : remainingMs(waitingUntil(item), now);
    const retryLeft = retry === undefined ? null : dueMs(retry.at, retry.afterMs, now);
    const blocked = state !== null && state.kind === "blocked" ? state.reason : null;
    const refusal = driftRefusal(item, page);
    const at = positionOf(siblings, itemId);

    const tabs: readonly TabItem<ArtifactKind>[] = kinds.map((kind) => ({
        key: kind,
        label: artifactLabel(kind),
    }));

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={path}
            closeLabel={copy.runs.review.close}
            width={drawerWidth}
            header={
                item === null ? undefined : (
                    <span className="flex shrink-0 items-center gap-1.5">
                        {at === 0 ? null : (
                            <span className="font-mono text-2xs text-ink-faint">
                                {copy.runs.review.position(at, siblings.length)}
                            </span>
                        )}
                        <StatusBadge tone={statusTone(item.status)} icon={statusIcon(item.status)}>
                            {statusLabel(item.status)}
                        </StatusBadge>
                    </span>
                )
            }
            footer={
                item === null ? undefined : (
                    <div className="flex w-full items-center justify-between gap-2">
                        <span className="flex shrink-0 items-center gap-1.5 text-2xs text-ink-faint">
                            <Kbd keys={["J", "K"]} />
                            {copy.runs.review.moveItems}
                        </span>
                        <div className="flex shrink-0 gap-2">
                            <Button
                                size="sm"
                                icon={OpenInNewIcon}
                                onClick={() => {
                                    onOpenPage(item.targetId);
                                }}
                            >
                                {copy.runs.review.openPage}
                            </Button>
                            <Button
                                size="sm"
                                variant="ghost"
                                icon={SmartToyIcon}
                                onClick={() => {
                                    askAgent(copy.agent.ask.runItem(path, item.runId, item.id));
                                }}
                            >
                                {copy.agent.askAbout}
                            </Button>
                            {blocked === null ? null : (
                                <Button
                                    size="sm"
                                    onClick={() => {
                                        onRerun(item.targetId);
                                    }}
                                >
                                    {copy.runs.rerunPage}
                                </Button>
                            )}
                            <Button
                                size="sm"
                                variant="primary"
                                icon={RestartAltIcon}
                                disabled={state === null || state.kind !== "ready"}
                                busy={retryStep.isPending}
                                title={
                                    state !== null && state.kind === "busy"
                                        ? copy.runs.retryBusy
                                        : blocked === null
                                          ? undefined
                                          : copy.runs.retryBlockedBody
                                }
                                onClick={() => {
                                    retryStep.mutate({ itemId: item.id });
                                }}
                            >
                                {copy.runs.retryStep}
                            </Button>
                        </div>
                    </div>
                )
            }
        >
            {item === null ? (
                <ReviewMissing narrowed={missing && narrowed} onClearFilter={onClearFilter} />
            ) : (
                <div className="flex h-full min-h-0 flex-col">
                    <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                        <span className="font-mono text-2xs text-ink-faint">
                            {stepText(steps, view?.step ?? item.currentStep)}
                        </span>
                        <span className="font-mono text-2xs text-ink-faint">
                            {copy.runs.review.attempt(item.attempts)}
                        </span>
                        {wake === null ? null : (
                            <span className="font-mono text-2xs text-info">
                                {copy.runs.step.waitingUntil(countdown(wake))}
                            </span>
                        )}
                        {retry === undefined || retryLeft === null ? null : (
                            <span className="font-mono text-2xs text-warn">
                                {copy.runs.step.retrying(retry.attempt, countdown(retryLeft))}
                            </span>
                        )}
                    </div>

                    {refusal !== null ? (
                        <DriftRefused
                            refusal={refusal}
                            resuming={resume.isPending}
                            onOpenPage={() => {
                                onOpenPage(item.targetId);
                            }}
                            onRerun={() => {
                                onRerun(item.targetId);
                            }}
                            onResume={() => {
                                resume.mutate({ runId: item.runId });
                            }}
                        />
                    ) : item.pauseReason === "" ? null : (
                        <div className="shrink-0 px-3 pt-2">
                            <Banner tone="warn" title={pauseReasonText(item.pauseReason)} />
                        </div>
                    )}
                    {item.error === "" ? null : (
                        <div className="shrink-0 px-3 pt-2">
                            <Banner tone="danger" title={item.error} />
                        </div>
                    )}
                    {blocked === null ? null : (
                        <div className="shrink-0 px-3 pt-2">
                            <Banner
                                tone="info"
                                title={retryBlockedText(blocked)}
                                body={copy.runs.retryBlockedBody}
                            />
                        </div>
                    )}

                    <Timeline entries={timeline} />

                    {listed.isPending ? (
                        <div className="p-3">
                            <SkeletonRows rows={5} label={copy.runs.review.loading} />
                        </div>
                    ) : kinds.length === 0 || active === null ? (
                        <ReviewEmpty step={stepText(steps, view?.step ?? item.currentStep)} />
                    ) : (
                        <div className="flex min-h-0 flex-1 flex-col">
                            <div className="flex h-8 shrink-0 items-center overflow-x-auto border-b border-hairline px-2">
                                <Tabs
                                    label={copy.runs.review.title}
                                    items={tabs}
                                    value={active}
                                    onValueChange={setActive}
                                />
                            </div>
                            <TabPanel label={artifactLabel(active)} active={true}>
                                <ArtifactPane
                                    key={`${item.id}:${active}`}
                                    itemId={item.id}
                                    kind={active}
                                    pageId={item.targetId}
                                    retentionDays={retentionDays}
                                />
                            </TabPanel>
                        </div>
                    )}
                </div>
            )}
        </Drawer>
    );
}
