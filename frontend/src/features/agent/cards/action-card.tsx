import type { ReactElement } from "react";
import { useMemo } from "react";

import { useFinding } from "../../../data/hooks/findings.js";
import { useTemplate } from "../../../data/hooks/templates.js";
import { useTools } from "../../../data/hooks/tools.js";
import type { Timestamp } from "../../../data/wire.js";
import { riskOf, toolLabel } from "../conversation/model/tools.js";
import type { CardBusy, CardKeys } from "./card.js";
import { ConfirmationCard } from "./card.js";
import { Lines, Parts } from "./lines.js";
import { isArgs, text } from "./model/card.js";
import { describeAction } from "./model/describe.js";
import { ownSummary } from "./model/summary.js";

const findingsTool = "findings";

function findingIdOf(tool: string, args: unknown): string | null {
    return tool === findingsTool && isArgs(args) ? text(args, "findingId") : null;
}

const templatesTool = "templates";
const templateOps: ReadonlySet<string> = new Set(["update", "override"]);

function templateIdOf(tool: string, args: unknown): string | null {
    if (tool !== templatesTool || !isArgs(args) || !templateOps.has(text(args, "op") ?? "")) {
        return null;
    }
    return text(args, "templateId");
}

export interface ActionCardProps {
    tool: string;
    args: unknown;
    summary: string;
    risk: string | null;
    status: string;
    createdAt: Timestamp;
    outcome: string | null;
    busy: CardBusy;
    focus?: boolean;
    keys?: CardKeys;
    onApprove?: () => void;
    onReject?: () => void;
    className?: string;
}

export function ActionCard({
    tool,
    args,
    summary,
    risk,
    status,
    createdAt,
    outcome,
    busy,
    focus,
    keys,
    onApprove,
    onReject,
    className,
}: ActionCardProps): ReactElement {
    const tools = useTools();
    const current = useTemplate(templateIdOf(tool, args));
    const read = useFinding(findingIdOf(tool, args));
    const listed = tools.data?.tools ?? [];
    const registered = listed.find((held) => held.name === tool) ?? null;
    const currentTemplate = current.data?.template.spec ?? null;
    const finding = read.data ?? null;
    const said = ownSummary(summary, registered?.description ?? null);
    const view = useMemo(
        () => describeAction(tool, args, registered?.schema ?? null, { currentTemplate, finding, summary: said }),
        [tool, args, registered, currentTemplate, finding, said],
    );

    return (
        <ConfirmationCard
            tool={tool}
            risk={risk ?? riskOf(tool, args, registered?.risk ?? "write")}
            status={status}
            createdAt={createdAt}
            outcome={outcome}
            title={<Parts parts={view.title} />}
            lines={<Lines lines={view.lines} />}
            summary={view.generic ? null : said}
            label={toolLabel(tool, args)}
            busy={busy}
            focus={focus}
            keys={keys}
            onApprove={onApprove}
            onReject={onReject}
            className={className}
        />
    );
}
