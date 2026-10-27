import type { ReactElement } from "react";
import { useRef, useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import {
    AddLinkIcon,
    Banner,
    Button,
    CountBadge,
    HubIcon,
    Screen,
    Segmented,
    SkeletonRows,
    TableRowsIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { GraphActions } from "./actions.js";
import { AdoptParentsDrawer } from "./actions/adopt-parents.js";
import { CardProposalsDrawer } from "./cards/drawer.js";
import { GraphEmpty } from "./empty.js";
import { Inspector } from "./inspector/panel.js";
import { GraphMapView } from "./map-view.js";
import { liftMore } from "./model/fold.js";
import type { GraphView } from "./model/params.js";
import { OutlineView } from "./outline/view.js";
import { GraphOverlays } from "./overlays.js";
import { ProposalQueue } from "./queue/sheet.js";
import { useGraphScreen } from "./screen-state.js";
import { GraphToolbar } from "./toolbar.js";

const viewOptions: readonly SegmentedOption<GraphView>[] = [
    { value: "map", label: copy.graph.views.map, icon: HubIcon },
    { value: "outline", label: copy.graph.views.outline, icon: TableRowsIcon },
];

export function GraphScreen(): ReactElement {
    const screen = useGraphScreen();
    const { siteId, entityId, query, graph, index, fold, session, patchSession } = screen;
    const host = useRef<HTMLDivElement>(null);
    const [adopting, setAdopting] = useState(false);

    const failure = graph.error === null ? null : react(graph.error);
    const empty = graph.data !== undefined && index.counts.total === 0;
    const connecting = screen.connectFrom === null ? undefined : index.byId.get(screen.connectFrom);
    const ready = !empty && !graph.isPending;

    const content = graph.isPending ? (
        <div className="p-4">
            <SkeletonRows rows={8} label={copy.graph.loading} />
        </div>
    ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
        <div className="p-4">
            <Banner
                tone="danger"
                title={failure.message}
                actions={
                    <Button size="sm" variant="secondary" onClick={() => void graph.refetch()}>
                        {copy.app.retry}
                    </Button>
                }
            />
        </div>
    ) : empty ? (
        <GraphEmpty
            onPropose={() => {
                screen.setProposing("pages");
            }}
            onImport={() => {
                void screen.navigate(`/s/${siteId}/import`);
            }}
        />
    ) : query.view === "outline" ? (
        <OutlineView
            siteId={siteId}
            index={index}
            rows={screen.rows}
            selectedId={entityId}
            matched={screen.matched}
            onSelect={screen.select}
            onPick={screen.pick}
            onToggle={screen.toggleNode}
            onLiftMore={(parentId, count) => {
                screen.setFold(liftMore(fold, parentId, count));
            }}
        />
    ) : (
        <GraphMapView
            siteId={siteId}
            index={index}
            rows={screen.rows}
            fold={fold}
            selectedId={entityId}
            matched={screen.matched}
            highlightEdgeId={screen.hoverEdge}
            pulse={screen.pulse}
            tint={screen.tint}
            revealVersion={screen.revealVersion}
            session={session}
            proof={query.proof}
            hostWidth={() => host.current?.clientWidth ?? 0}
            onPatchSession={patchSession}
            onFold={screen.setFold}
            onReveal={screen.revealEntity}
            onSelect={screen.select}
            onPick={screen.pick}
            onToggle={screen.toggleNode}
            onCreateChild={(parentId) => {
                screen.setCreating({ parentId });
            }}
            onConnect={screen.startConnect}
            onDelete={screen.setDeleting}
            onReparent={(childId, parentId) => {
                screen.setMoving({ childId, parentId });
            }}
        />
    );

    return (
        <Screen
            title={copy.graph.title}
            badge={graph.data === undefined ? undefined : <CountBadge tone="muted" count={index.counts.total} />}
            tabs={
                <Segmented
                    label={copy.graph.views.label}
                    value={query.view}
                    options={viewOptions}
                    onValueChange={(view) => {
                        screen.change({ ...query, view });
                    }}
                />
            }
            actions={
                <GraphActions
                    index={index}
                    selectedId={entityId}
                    connecting={screen.connectFrom !== null}
                    reviewing={session.queue}
                    recomputing={screen.recompute.isPending}
                    cardProposals={screen.proposedCards.length}
                    onReviewCards={() => {
                        patchSession({ cards: true, cardFocus: null });
                    }}
                    onCreate={() => {
                        screen.setCreating({ parentId: entityId });
                    }}
                    onConnect={() => {
                        if (entityId !== null) {
                            screen.startConnect(entityId);
                        }
                    }}
                    onStopConnect={screen.stopConnect}
                    onReview={() => {
                        patchSession({ queue: !session.queue });
                    }}
                    onProposeFromPages={() => {
                        screen.setProposing("pages");
                    }}
                    onProposeFromKeywords={() => {
                        screen.setProposing("keywords");
                    }}
                    onProposeRelated={() => {
                        screen.setProposing("related");
                    }}
                    onRecompute={screen.recomputeScores}
                    onAdopt={() => {
                        setAdopting(true);
                    }}
                />
            }
            toolbar={
                ready ? (
                    <GraphToolbar
                        index={index}
                        query={query}
                        proofBusy={query.proof && screen.audit.isPending}
                        focusSearch={screen.focusSearch}
                        onChange={screen.change}
                        onPick={screen.revealEntity}
                    />
                ) : undefined
            }
            variant="full"
            right={
                ready ? (
                    <Inspector
                        siteId={siteId}
                        index={index}
                        selectedId={entityId}
                        onSelect={screen.select}
                        onReveal={screen.revealEntity}
                        onLens={screen.onLens}
                        onConnect={screen.startConnect}
                        onDelete={screen.setDeleting}
                        onPlanPage={screen.setPlanning}
                        onRecompute={screen.recomputeScores}
                        recomputing={screen.recompute.isPending}
                        audit={screen.auditPages}
                        proposed={screen.waitingCards}
                        onReviewCard={(id) => {
                            patchSession({ cards: true, cardFocus: id });
                        }}
                    />
                ) : undefined
            }
        >
            <div
                className="flex h-full min-h-0 flex-col"
                onKeyDown={(event) => {
                    if ((event.ctrlKey || event.metaKey) && (event.key === "f" || event.key === "F")) {
                        event.preventDefault();
                        screen.setFocusSearch((held) => held + 1);
                    }
                }}
            >
                {connecting === undefined ? null : (
                    <div className="border-b border-hairline px-4 py-2">
                        <Banner
                            tone="info"
                            icon={AddLinkIcon}
                            title={copy.graph.connect.picking(connecting.name)}
                            actions={
                                <Button size="sm" variant="ghost" onClick={screen.stopConnect}>
                                    {copy.graph.connect.stop}
                                </Button>
                            }
                        />
                    </div>
                )}
                <div ref={host} className="relative flex min-h-0 flex-1 flex-col">
                    {content}
                    {ready ? (
                        <ProposalQueue
                            index={index}
                            open={session.queue}
                            height={session.queueHeight}
                            onHeightChange={(next) => {
                                patchSession({ queueHeight: next });
                            }}
                            onClose={() => {
                                patchSession({ queue: false });
                                screen.setHoverEdge(null);
                            }}
                            onHover={screen.setHoverEdge}
                            onReveal={screen.revealEntity}
                        />
                    ) : null}
                </div>
            </div>
            <GraphOverlays
                siteId={siteId}
                index={index}
                entityIndex={screen.entityIndex}
                selectedId={entityId}
                creating={screen.creating}
                connectFrom={screen.connectFrom}
                connectTo={screen.connectTo}
                deleting={screen.deleting}
                moving={screen.moving}
                proposing={screen.proposing}
                planning={screen.planning}
                onCreatingChange={screen.setCreating}
                onStopConnect={screen.stopConnect}
                onDeletingChange={screen.setDeleting}
                onMovingChange={screen.setMoving}
                onProposingChange={screen.setProposing}
                onPlanningChange={screen.setPlanning}
                onSelect={screen.select}
                onReviewQueue={screen.openQueue}
            />
            <CardProposalsDrawer
                siteId={siteId}
                open={session.cards}
                focus={session.cardFocus}
                onOpenChange={(next) => {
                    patchSession(next ? { cards: true } : { cards: false, cardFocus: null });
                }}
                onReveal={screen.revealEntity}
            />
            {adopting ? <AdoptParentsDrawer open={true} siteId={siteId} under={null} onOpenChange={setAdopting} /> : null}
        </Screen>
    );
}
