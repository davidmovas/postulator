import type { ReactElement } from "react";
import { useMemo } from "react";

import { useTemplate } from "../../../data/hooks/templates.js";
import { useTools } from "../../../data/hooks/tools.js";
import type { Timestamp } from "../../../data/wire.js";
import type { CardBusy } from "./card.js";
import { ConfirmationCard } from "./card.js";
import { Lines, Parts } from "./lines.js";
import { isArgs, text } from "./model/card.js";
import { describeAction } from "./model/describe.js";

function templateIdOf(tool: string, args: unknown): string | null {
    if (!isArgs(args)) {
        return null;
    }
    switch (tool) {
        case "templates_update":
            return text(args, "id");
        case "templates_set_override":
            return text(args, "templateId");
        default:
            return null;
    }
}

export interface ActionCardProps {
    tool: string;
    args: unknown;
    risk: string | null;
    status: string;
    createdAt: Timestamp;
    outcome: string | null;
    busy: CardBusy;
    focus?: boolean;
    onApprove?: () => void;
    onReject?: () => void;
    className?: string;
}

export function ActionCard({ tool, args, risk, status, createdAt, outcome, busy, focus, onApprove, onReject, className }: ActionCardProps): ReactElement {
    const tools = useTools();
    const current = useTemplate(templateIdOf(tool, args));
    const listed = tools.data?.tools ?? [];
    const registered = listed.find((held) => held.name === tool) ?? null;
    const currentTemplate = current.data?.template.spec ?? null;
    const view = useMemo(
        () => describeAction(tool, args, registered?.schema ?? null, { currentTemplate }),
        [tool, args, registered, currentTemplate],
    );

    return (
        <ConfirmationCard
            tool={tool}
            risk={risk ?? registered?.risk ?? "write"}
            status={status}
            createdAt={createdAt}
            outcome={outcome}
            title={<Parts parts={view.title} />}
            lines={<Lines lines={view.lines} />}
            busy={busy}
            focus={focus}
            onApprove={onApprove}
            onReject={onReject}
            className={className}
        />
    );
}
