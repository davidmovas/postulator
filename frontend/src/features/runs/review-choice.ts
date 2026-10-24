import { useEffect, useMemo, useState } from "react";

import { useArtifact } from "../../data/hooks/runs.js";
import { decodeArtifact, editPlanView } from "./artifacts.js";
import type { ReviewTicks } from "./review-ticks.js";
import { offeredKeys, toggled } from "./review-ticks.js";
import { artifactEditPlan } from "./statuses.js";

export interface ReviewChoice extends ReviewTicks {
    onToggle: (key: string) => void;
}

export function useReviewChoice(itemId: string, reviewing: boolean): ReviewChoice | null {
    const held = useArtifact(reviewing && itemId !== "" ? itemId : null, artifactEditPlan);
    const [unticked, setUnticked] = useState<ReadonlySet<string>>(() => new Set());

    useEffect(() => {
        setUnticked(new Set());
    }, [itemId]);

    const content = held.data?.artifact.content;
    const offered = useMemo(
        () => (content === undefined ? [] : offeredKeys(editPlanView(decodeArtifact(content)))),
        [content],
    );

    if (!reviewing || offered.length === 0) {
        return null;
    }
    return {
        offered,
        unticked,
        onToggle: (key) => {
            setUnticked((was) => toggled(was, key));
        },
    };
}
