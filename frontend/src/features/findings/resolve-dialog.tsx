import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { errorMessageOf, failure } from "../../data/errors.js";
import { useExplanation, useResolveFinding } from "../../data/hooks/findings.js";
import type { ExplainRequest, Finding } from "../../data/types.js";
import { Banner, CheckIcon, Dialog, Skeleton } from "../../ui/index.js";
import { CardBody } from "../agent/cards/body.js";
import { Lines, Parts } from "../agent/cards/lines.js";
import { describeAction } from "../agent/cards/model/describe.js";
import { optionLabel } from "./labels.js";
import { rowCandidates } from "./model/subject.js";
import { RowPagePick, TypePick } from "./pickers.js";

const said = copy.findings.resolve;

const findingsTool = "findings";
const mapToPage = "map_to_page";
const setType = "set_type";
const conflict = "CONFLICT";

export interface Resolving {
    finding: Finding;
    option: string;
}

export interface Picks {
    pageId?: string;
    pageTypeId?: string;
}

export function picksOf(option: string, pageId: string | null, pageTypeId: string | null): Picks {
    if (option === mapToPage && pageId !== null) {
        return { pageId };
    }
    if (option === setType && pageTypeId !== null) {
        return { pageTypeId };
    }
    return {};
}

export function pickRefusal(option: string, picks: Picks): string | null {
    if (option === mapToPage && picks.pageId === undefined) {
        return said.pageFirst;
    }
    if (option === setType && picks.pageTypeId === undefined) {
        return said.typeFirst;
    }
    return null;
}

export interface ResolveDialogProps {
    siteId: string;
    resolving: Resolving | null;
    onClose: () => void;
    onGone: () => void;
}

export function ResolveDialog({ siteId, resolving, onClose, onGone }: ResolveDialogProps): ReactElement {
    const [pageId, setPageId] = useState<string | null>(null);
    const [pageTypeId, setPageTypeId] = useState<string | null>(null);
    const resolve = useResolveFinding();
    const resetResolve = resolve.reset;
    const finding = resolving?.finding ?? null;
    const option = resolving?.option ?? "";

    useEffect(() => {
        setPageId(null);
        setPageTypeId(null);
        resetResolve();
    }, [resolving, resetResolve]);

    const picks = picksOf(option, pageId, pageTypeId);
    const refusal = pickRefusal(option, picks);
    const asked: ExplainRequest | null =
        finding === null || refusal !== null ? null : { siteId, findingIds: [finding.id], option, ...picks };
    const explanation = useExplanation(asked);
    const view = useMemo(
        () =>
            finding === null
                ? null
                : describeAction(findingsTool, { op: "resolve", findingId: finding.id, option, ...picks }, null, { currentTemplate: null, finding }),
        [finding, option, picks.pageId, picks.pageTypeId],
    );
    const thrown = resolve.error ?? explanation.error;
    const gone = thrown !== null && failure(thrown).code === conflict;
    const problem = gone ? said.gone : errorMessageOf(thrown);

    return (
        <Dialog
            open={finding !== null}
            onOpenChange={(open) => {
                if (!open) {
                    onClose();
                }
            }}
            title={optionLabel(option)}
            description={said.body}
            confirmLabel={said.confirm}
            cancelLabel={said.cancel}
            icon={CheckIcon}
            busy={resolve.isPending}
            refusal={refusal ?? (gone ? said.gone : null)}
            onConfirm={() => {
                if (finding === null || refusal !== null) {
                    return;
                }
                resolve.mutate(
                    { siteId, findingId: finding.id, option, ...picks },
                    {
                        onSuccess: onClose,
                        onError: (error) => {
                            if (failure(error).code === conflict) {
                                onGone();
                            }
                        },
                    },
                );
            }}
        >
            {finding !== null && option === setType ? <TypePick siteId={siteId} value={pageTypeId} onChange={setPageTypeId} /> : null}
            {finding !== null && option === mapToPage ? (
                <RowPagePick siteId={siteId} candidates={rowCandidates(finding.subject)} value={pageId} onChange={setPageId} />
            ) : null}
            {view === null ? null : (
                <div className="flex flex-col gap-2.5 rounded-lg border border-hairline bg-panel p-3" data-resolve-card={true}>
                    <CardBody
                        tool={findingsTool}
                        tone="warn"
                        title={<Parts parts={view.title} />}
                        lines={<Lines lines={view.lines} />}
                        summary={explanation.data?.sentence ?? null}
                    />
                    {asked !== null && explanation.isPending ? <Skeleton width="70%" height={12} /> : null}
                </div>
            )}
            {problem === null ? null : <Banner tone={gone ? "warn" : "danger"} title={problem} />}
        </Dialog>
    );
}
