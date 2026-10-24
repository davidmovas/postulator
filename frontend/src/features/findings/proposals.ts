import { useState } from "react";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useCollectFindingIds, useResolveFindings } from "../../data/hooks/findings.js";
import { pushToast } from "../../data/toasts.js";
import type { FindingFilter, PreviewFromPagesRequest } from "../../data/types.js";
import { batches } from "./model/bulk.js";

const conflict = "CONFLICT";

export interface BulkProposal {
    gathering: boolean;
    propose: (option: string) => void;
}

export interface BulkProposalArgs {
    siteId: string;
    code: string;
    filter: FindingFilter;
    onSeed: (seed: PreviewFromPagesRequest) => void;
    onGone: () => void;
}

export function useBulkProposal({ siteId, code, filter, onSeed, onGone }: BulkProposalArgs): BulkProposal {
    const collect = useCollectFindingIds();
    const resolveMany = useResolveFindings();
    const [gathering, setGathering] = useState(false);

    const gather = async (option: string): Promise<void> => {
        setGathering(true);
        try {
            const ids = await collect.mutateAsync(filter);
            const seen = new Set<string>();
            const pageIds: string[] = [];
            let answered: PreviewFromPagesRequest | null = null;
            for (const batch of batches(ids)) {
                const response = await resolveMany.mutateAsync({ siteId, code, option, findingIds: batch });
                const proposals = response.proposals ?? null;
                if (proposals === null) {
                    continue;
                }
                answered = proposals;
                for (const pageId of proposals.pageIds ?? []) {
                    if (!seen.has(pageId)) {
                        seen.add(pageId);
                        pageIds.push(pageId);
                    }
                }
            }
            if (answered !== null) {
                onSeed({ ...answered, pageIds });
            }
        } catch (error) {
            if (failure(error).code === conflict) {
                pushToast("warning", copy.findings.resolve.gone);
                onGone();
            }
        } finally {
            setGathering(false);
        }
    };

    return {
        gathering,
        propose: (option) => {
            void gather(option);
        },
    };
}
