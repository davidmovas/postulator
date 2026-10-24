import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { Finding, FindingChoice } from "../../data/types.js";
import { Banner, Button, EmptyState, SkeletonRows, TaskAltIcon, VirtualRows } from "../../ui/index.js";
import { FindingRow, findingRowHeight } from "./row.js";

const said = copy.findings;

export interface FindingListProps {
    findings: readonly Finding[];
    pending: boolean;
    error: unknown;
    open: boolean;
    narrowed: boolean;
    hasMore: boolean;
    onMore: () => void;
    onRetry: () => void;
    onReset: () => void;
    onOpenPage: (pageId: string) => void;
    onChoose: (finding: Finding, option: FindingChoice) => void;
}

export function FindingList({
    findings,
    pending,
    error,
    open,
    narrowed,
    hasMore,
    onMore,
    onRetry,
    onReset,
    onOpenPage,
    onChoose,
}: FindingListProps): ReactElement {
    const failure = error === null || error === undefined ? null : react(error);
    if (pending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={8} label={said.loading} />
            </div>
        );
    }
    if (failure !== null && failure.kind !== "silent" && failure.kind !== "unlock") {
        return (
            <div className="p-4">
                <Banner
                    tone="danger"
                    title={failure.message}
                    actions={
                        <Button size="sm" variant="secondary" onClick={onRetry}>
                            {copy.app.retry}
                        </Button>
                    }
                />
            </div>
        );
    }
    if (findings.length === 0) {
        return (
            <div className="p-4">
                <EmptyState
                    icon={TaskAltIcon}
                    title={open ? said.empty.openTitle : said.empty.settledTitle}
                    body={open ? said.empty.openBody : said.empty.settledBody}
                    actions={
                        narrowed ? (
                            <Button size="sm" variant="secondary" onClick={onReset}>
                                {said.empty.filtered}
                            </Button>
                        ) : undefined
                    }
                />
            </div>
        );
    }
    return (
        <VirtualRows
            count={findings.length}
            rowHeight={findingRowHeight}
            onReachEnd={hasMore ? onMore : undefined}
            row={(index) => {
                const finding = findings[index];
                return finding === undefined ? null : (
                    <FindingRow finding={finding} onOpenPage={onOpenPage} onChoose={onChoose} />
                );
            }}
        />
    );
}
