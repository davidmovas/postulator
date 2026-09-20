import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { useJudgePage } from "../../../data/hooks/reports.js";
import { Button, GavelIcon, SectionLabel } from "../../../ui/index.js";
import type { JudgeView } from "../artifacts.js";
import { judgeView } from "../artifacts.js";

export interface JudgePaneProps {
    payload: unknown;
    pageId: string;
}

export function JudgePane({ payload, pageId }: JudgePaneProps): ReactElement {
    const stored = judgeView(payload);
    const judge = useJudgePage();
    const [fresh, setFresh] = useState<JudgeView | null>(null);
    const view = fresh ?? stored;

    return (
        <div className="flex flex-col gap-3 p-3">
            {view === null ? (
                <p className="text-xs text-ink-dim">{copy.runs.review.judge.unavailable}</p>
            ) : (
                <>
                    <div className="flex items-end gap-2">
                        <span className="font-mono text-2xl leading-none text-ink">{view.score.toFixed(2)}</span>
                        <span className="pb-0.5 text-2xs text-ink-faint">{copy.runs.review.judge.outOf}</span>
                    </div>
                    <div>
                        <SectionLabel>{copy.runs.review.judge.issues}</SectionLabel>
                        {view.issues.length === 0 ? (
                            <p className="pt-1 text-xs text-ink-dim">{copy.runs.review.judge.none}</p>
                        ) : (
                            <ul className="flex flex-col gap-1 pt-1">
                                {view.issues.map((issue) => (
                                    <li key={issue} className="text-xs text-ink-soft">
                                        {issue}
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                    <div>
                        <SectionLabel>{copy.runs.review.judge.suggestions}</SectionLabel>
                        {view.suggestions.length === 0 ? (
                            <p className="pt-1 text-xs text-ink-dim">{copy.runs.review.judge.none}</p>
                        ) : (
                            <ul className="flex flex-col gap-1 pt-1">
                                {view.suggestions.map((suggestion) => (
                                    <li key={suggestion} className="text-xs text-ink-soft">
                                        {suggestion}
                                    </li>
                                ))}
                            </ul>
                        )}
                    </div>
                </>
            )}
            <div className="flex flex-col gap-1 border-t border-hairline pt-3">
                <Button
                    size="sm"
                    icon={GavelIcon}
                    busy={judge.isPending}
                    disabled={pageId === ""}
                    title={copy.runs.review.judge.rejudgeHint}
                    onClick={() => {
                        judge.mutate(
                            { pageId },
                            {
                                onSuccess: (answered) => {
                                    setFresh(judgeView(answered.report));
                                },
                            },
                        );
                    }}
                >
                    {judge.isPending ? copy.runs.review.judge.rejudging : copy.runs.review.judge.rejudge}
                </Button>
                {judge.data === undefined ? null : (
                    <p className="text-2xs text-ok">{copy.runs.review.judge.rejudged(judge.data.tokens)}</p>
                )}
            </div>
        </div>
    );
}
