import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { errorMessageOf, failure } from "../../data/errors.js";
import { useExplanation, useFindingIds, useResolveFindings } from "../../data/hooks/findings.js";
import type { FindingFilter } from "../../data/types.js";
import { Banner, Dialog, Skeleton, TaskAltIcon } from "../../ui/index.js";
import { findingCodeLabel } from "../pages/model/labels.js";
import { optionLabel } from "./labels.js";
import { batches } from "./model/bulk.js";
import { TypePick } from "./pickers.js";
import { pickRefusal, picksOf } from "./resolve-dialog.js";

const said = copy.findings;

const conflict = "CONFLICT";

export interface BulkDialogProps {
    siteId: string;
    code: string;
    filter: FindingFilter;
    option: string | null;
    onClose: () => void;
    onGone: () => void;
}

export function BulkDialog({ siteId, code, filter, option, onClose, onGone }: BulkDialogProps): ReactElement {
    const [pageTypeId, setPageTypeId] = useState<string | null>(null);
    const [busy, setBusy] = useState(false);
    const [thrown, setThrown] = useState<unknown>(null);
    const ids = useFindingIds(filter, option !== null);
    const resolveMany = useResolveFindings();
    const chosen = option ?? "";
    const collected = ids.data ?? [];

    useEffect(() => {
        setPageTypeId(null);
        setThrown(null);
    }, [option]);

    const picks = picksOf(chosen, null, pageTypeId);
    const missing = pickRefusal(chosen, picks);
    const explanation = useExplanation(
        option === null || collected.length === 0 || missing !== null ? null : { siteId, findingIds: collected, option, ...picks },
    );
    const refusal = ids.isPending ? said.bulk.collecting : (missing ?? null);
    const failed = thrown ?? explanation.error ?? ids.error;
    const gone = failed !== null && failure(failed).code === conflict;
    const problem = gone ? said.resolve.gone : errorMessageOf(failed);

    const confirm = async (): Promise<void> => {
        if (option === null || refusal !== null) {
            return;
        }
        setBusy(true);
        setThrown(null);
        try {
            for (const batch of batches(collected)) {
                await resolveMany.mutateAsync({ siteId, code, option, findingIds: batch, ...picks });
            }
            onClose();
        } catch (error) {
            setThrown(error);
            if (failure(error).code === conflict) {
                onGone();
            }
        } finally {
            setBusy(false);
        }
    };

    return (
        <Dialog
            open={option !== null}
            onOpenChange={(open) => {
                if (!open) {
                    onClose();
                }
            }}
            title={said.bulk.title(optionLabel(chosen))}
            description={ids.isPending ? said.bulk.collecting : said.bulk.body(collected.length)}
            confirmLabel={said.bulk.confirm(collected.length)}
            cancelLabel={said.resolve.cancel}
            icon={TaskAltIcon}
            busy={busy}
            refusal={refusal}
            onConfirm={() => {
                void confirm();
            }}
        >
            {chosen === "set_type" ? <TypePick siteId={siteId} value={pageTypeId} onChange={setPageTypeId} /> : null}
            <div className="flex flex-col gap-1.5 rounded-lg border border-hairline bg-panel p-3" data-bulk-card={true}>
                <span className="text-sm font-semibold text-ink">{findingCodeLabel(code)}</span>
                {explanation.data === undefined ? (
                    explanation.isPending && collected.length > 0 && missing === null ? <Skeleton width="70%" height={12} /> : null
                ) : (
                    <p className="border-l-2 border-hairline pl-2.5 text-xs leading-snug text-ink-soft">{explanation.data.sentence}</p>
                )}
            </div>
            {problem === null ? null : <Banner tone={gone ? "warn" : "danger"} title={problem} />}
        </Dialog>
    );
}
