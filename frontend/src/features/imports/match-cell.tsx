import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewPage } from "../../data/types.js";
import { Select, TableCell } from "../../ui/index.js";
import type { Choices } from "./choices.js";
import { chosenFor, needsUrl, pickable } from "./choices.js";
import { matchRuleLabel, matchRulePhrase } from "./labels.js";

const said = copy.imports.preview;

const mapped = "mapped";

function pickedOf(choices: Choices, page: PreviewPage): string | null {
    const chosen = chosenFor(choices, page);
    if (chosen !== null || page.matchedBy !== mapped) {
        return chosen;
    }
    return (page.candidates ?? []).find((candidate) => candidate.path === page.path)?.pageId ?? null;
}

export interface MatchCellProps {
    page: PreviewPage;
    choices: Choices;
    onPick: (rowKey: string, pageId: string) => void;
}

export function MatchCell({ page, choices, onPick }: MatchCellProps): ReactElement {
    const held = page.matchedBy ?? "";
    const rule = matchRuleLabel(held);
    if (pickable(page)) {
        const rowKey = page.rowKey ?? "";
        return (
            <TableCell title={held === mapped ? said.matchedTooltip(matchRulePhrase(held)) : said.pickTooltip(matchRulePhrase(held))}>
                <Select
                    size="sm"
                    value={pickedOf(choices, page)}
                    placeholder={said.pick}
                    aria-label={said.pickLabel(page.path)}
                    options={(page.candidates ?? []).map((candidate) => ({
                        value: candidate.pageId,
                        label: said.candidate(candidate.path, candidate.title),
                    }))}
                    onValueChange={(pageId) => {
                        onPick(rowKey, pageId);
                    }}
                />
            </TableCell>
        );
    }
    if (needsUrl(page)) {
        return (
            <TableCell muted={true} title={said.needsUrlTooltip}>
                {said.needsUrl}
            </TableCell>
        );
    }
    return (
        <TableCell muted={true} title={rule === "" ? undefined : said.matchedTooltip(matchRulePhrase(held))}>
            {rule}
        </TableCell>
    );
}
