import type { ReactElement } from "react";
import { useEffect } from "react";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure } from "../../../data/errors.js";
import { useCardProposals } from "../../../data/hooks/graph.js";
import { Banner, Button, CountBadge, Drawer, EditNoteIcon, EmptyState, SkeletonRows } from "../../../ui/index.js";
import { CardProposalItem } from "./proposal.js";

const said = copy.graph.cards;

const drawerWidth = 720;

export interface CardProposalsDrawerProps {
    siteId: string;
    open: boolean;
    focus: string | null;
    onOpenChange: (open: boolean) => void;
    onReveal: (entityId: string) => void;
}

export function CardProposalsDrawer({ siteId, open, focus, onOpenChange, onReveal }: CardProposalsDrawerProps): ReactElement | null {
    const listed = useCardProposals(open ? siteId : null);
    const proposals = flatten(listed.data?.pages);
    const loaded = proposals.length;

    useEffect(() => {
        if (!open || focus === null || loaded === 0) {
            return;
        }
        for (const node of document.querySelectorAll<HTMLElement>("[data-card-proposal]")) {
            if (node.dataset["cardProposal"] === focus) {
                node.scrollIntoView({ block: "nearest" });
            }
        }
    }, [open, focus, loaded]);

    if (!open) {
        return null;
    }

    return (
        <Drawer
            open={true}
            onOpenChange={onOpenChange}
            title={said.title}
            description={said.description}
            closeLabel={said.close}
            width={drawerWidth}
            header={loaded === 0 ? null : <CountBadge tone="info" count={loaded} />}
        >
            {listed.isPending ? (
                <div className="p-3">
                    <SkeletonRows rows={6} label={said.loading} />
                </div>
            ) : listed.isError && loaded === 0 ? (
                <div className="p-3">
                    <Banner tone="danger" title={failure(listed.error).message} />
                </div>
            ) : loaded === 0 ? (
                <div className="p-6">
                    <EmptyState icon={EditNoteIcon} title={said.noneTitle} body={said.noneBody} />
                </div>
            ) : (
                <div className="flex flex-col gap-3 p-3">
                    <p className="text-xs text-ink-dim">{said.waiting(loaded, listed.hasNextPage)}</p>
                    <ul className="flex flex-col gap-3">
                        {proposals.map((proposal) => (
                            <CardProposalItem
                                key={proposal.findingId}
                                siteId={siteId}
                                proposal={proposal}
                                focused={proposal.entityId === focus}
                                onReveal={(entityId) => {
                                    onOpenChange(false);
                                    onReveal(entityId);
                                }}
                            />
                        ))}
                    </ul>
                    {listed.hasNextPage ? (
                        <Button
                            variant="ghost"
                            className="self-center"
                            busy={listed.isFetchingNextPage}
                            onClick={() => void listed.fetchNextPage()}
                        >
                            {said.loadMore}
                        </Button>
                    ) : null}
                </div>
            )}
        </Drawer>
    );
}
