import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { AddLinkIcon, Banner, Checkbox } from "../../ui/index.js";
import type { MustLinkRefusal } from "./start-directions.js";

const said = copy.runs.start.mustLink;

export interface MustLinkNoticeProps {
    refusal: MustLinkRefusal | null;
    addEdges: boolean;
    asked: boolean;
    busy: boolean;
    onAddEdges: (on: boolean) => void;
}

export function MustLinkNotice({
    refusal,
    addEdges,
    asked,
    busy,
    onAddEdges,
}: MustLinkNoticeProps): ReactElement | null {
    if (refusal === null && !(addEdges && asked)) {
        return null;
    }
    const offer = (
        <div className="flex flex-col gap-1">
            <Checkbox
                label={said.addEdges}
                checked={addEdges}
                disabled={busy}
                data-run-add-edges={true}
                onChange={(event) => {
                    onAddEdges(event.currentTarget.checked);
                }}
            />
            <span className="text-2xs text-ink-faint">{said.addEdgesHint}</span>
        </div>
    );
    if (refusal === null) {
        return (
            <div
                className="flex flex-col gap-1 rounded-md border border-hairline bg-inset px-2.5 py-2"
                data-run-must-link="adding"
            >
                {offer}
                <span className="text-xs text-ink-soft">{said.adding}</span>
            </div>
        );
    }
    return (
        <div data-run-must-link="refused">
            <Banner tone="warn" icon={AddLinkIcon} title={refusal.message} body={addEdges ? undefined : offer} />
        </div>
    );
}
