import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { editPlanView } from "../artifacts.js";
import type { ReviewChoice } from "../review-choice.js";
import type { Ticks } from "./diff.js";
import { BlockDiff, FieldDiff } from "./diff.js";
import { SiteRegions } from "./regions.js";
import { Unreadable } from "./shared.js";

const said = copy.runs.review;

export interface EditPlanPaneProps {
    payload: unknown;
    choice: ReviewChoice | null;
}

export function EditPlanPane({ payload, choice }: EditPlanPaneProps): ReactElement {
    const view = editPlanView(payload);
    if (view === null) {
        return <Unreadable />;
    }
    const ticks: Ticks | null =
        choice === null
            ? null
            : {
                  ticked: (key) => !choice.unticked.has(key),
                  onToggle: choice.onToggle,
              };

    return (
        <div className="flex flex-col gap-2 pb-3" title={said.noDiff}>
            <p className="px-3 pt-2 text-xs text-ink-dim">{said.plan.noImages}</p>
            {view.pages.length === 0 ? (
                <p className="px-3 text-xs text-ink-dim">{said.plan.none}</p>
            ) : (
                view.pages.map((page) => (
                    <section key={page.pageId} aria-label={page.path} className="flex flex-col gap-2 border-t border-hairline pt-2">
                        <span className="truncate px-3 font-mono text-xs text-ink">{page.path}</span>
                        <FieldDiff fields={page.fields} ticks={ticks} />
                        <BlockDiff
                            blocks={page.blocks}
                            tick={ticks === null || page.body === "" ? null : { key: page.body, ticks }}
                        />
                        {page.pageId === "" ? null : <SiteRegions pageId={page.pageId} />}
                    </section>
                ))
            )}
        </div>
    );
}
