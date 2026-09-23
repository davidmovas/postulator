import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { useArtifacts, useResumeRun, useRetryStep } from "../../data/hooks/runs.js";
import { useSetting } from "../../data/hooks/settings.js";
import type { Page, RunItem } from "../../data/types.js";
import type { ArtifactKind } from "../../generated/vocab.js";
import type { TabItem } from "../../ui/index.js";
import { Banner, Drawer, SkeletonRows, StatusBadge, TabPanel, Tabs } from "../../ui/index.js";
import { dueMs, remainingMs, retryState, waitingUntil } from "./authority.js";
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
import { DriftRefused, ReviewActions, ReviewEmpty, ReviewMeta, ReviewMissing, Timeline } from "./review-states.js";
import { artifactBodyHtml, retentionDaysKey } from "./statuses.js";
import { stepText } from "./step-cell.js";

const drawerWidth = 688;

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
                    <ReviewActions
                        path={path}
                        runId={item.runId}
                        itemId={item.id}
                        pageId={item.targetId}
                        blocked={blocked}
                        state={state}
                        busy={retryStep.isPending}
                        onOpenPage={onOpenPage}
                        onRerun={onRerun}
                        onRetry={() => {
                            retryStep.mutate({ itemId: item.id });
                        }}
                    />
                )
            }
        >
            {item === null ? (
                <ReviewMissing narrowed={missing && narrowed} onClearFilter={onClearFilter} />
            ) : (
                <div className="flex h-full min-h-0 flex-col">
                    <ReviewMeta
                        step={stepText(steps, view?.step ?? item.currentStep)}
                        attempts={item.attempts}
                        wake={wake}
                        retry={retry}
                        retryLeft={retryLeft}
                    />

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
