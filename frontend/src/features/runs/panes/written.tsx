import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useRun } from "../../../data/hooks/runs.js";
import { Button, HistoryIcon } from "../../../ui/index.js";
import { fieldLabel } from "../../settings/types/labels.js";
import type { PublishView } from "../artifacts.js";
import { revertState } from "../authority.js";
import { RevertFieldsDialog } from "../revert-fields.js";
import { bodyKeyOf, picked, revertableKeys } from "../revert-pick.js";
import { toggled } from "../review-ticks.js";
import type { Ticks } from "./diff.js";
import { BlockDiff, Declined, FieldDiff } from "./diff.js";

const said = copy.runs.revertFields;

export interface PublishDiffProps {
    view: PublishView;
    pageId: string;
    runId: string;
}

function refusalOf(view: PublishView, chosen: number, blocked: string | null): string | undefined {
    if (view.created) {
        return said.created;
    }
    if (blocked !== null) {
        return blocked;
    }
    return chosen === 0 ? said.none : undefined;
}

export function PublishDiff({ view, pageId, runId }: PublishDiffProps): ReactElement {
    const run = useRun(runId === "" ? null : runId).data?.run;
    const [ticked, setTicked] = useState<ReadonlySet<string>>(() => new Set());
    const [confirming, setConfirming] = useState(false);

    const offered = pageId === "" ? [] : revertableKeys(view);
    const chosen = picked(offered, ticked);
    const state = run === undefined ? null : revertState(run);
    const blocked =
        state === null || state.kind === "ready"
            ? null
            : state.reason === "in_flight"
              ? copy.runs.revertRunning
              : copy.runs.revertOfARevert;
    const ticks: Ticks = {
        ticked: (key) => ticked.has(key),
        onToggle: (key) => {
            setTicked((was) => toggled(was, key));
        },
        offers: (key) => offered.includes(key),
    };
    const body = bodyKeyOf(view);
    const refusal = refusalOf(view, chosen.length, blocked);

    return (
        <>
            <FieldDiff fields={view.fields} empty={copy.runs.review.diff.noFields} ticks={ticks} />
            <Declined keys={view.declined} />
            <BlockDiff
                blocks={view.blocks}
                empty={copy.runs.review.diff.noBlocks}
                tick={offered.includes(body) ? { key: body, ticks } : null}
            />
            {pageId === "" ? null : (
                <div className="flex justify-end px-3">
                    <Button
                        size="sm"
                        icon={HistoryIcon}
                        data-publish-revert={true}
                        disabled={run === undefined || refusal !== undefined}
                        title={refusal}
                        onClick={() => {
                            setConfirming(true);
                        }}
                    >
                        {said.action(chosen.length)}
                    </Button>
                </div>
            )}
            <RevertFieldsDialog
                open={confirming}
                onOpenChange={setConfirming}
                runId={runId}
                pageId={pageId}
                siteId={run?.siteId ?? ""}
                fields={chosen}
                description={said.body(chosen.map((key) => fieldLabel(key)).join(", "))}
            />
        </>
    );
}
