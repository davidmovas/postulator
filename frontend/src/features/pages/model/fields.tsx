import type { ReactElement } from "react";
import { useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import type { ModelField } from "../../../data/types.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import { Button, HistoryIcon, Panel, PanelHeader, StatusBadge } from "../../../ui/index.js";
import { RevertFieldsDialog } from "../../runs/revert-fields.js";
import { fieldLabel } from "../../settings/types/labels.js";
import { Note } from "./facts.js";

const said = copy.pages.model.fields;

export interface WrittenFieldsPaneProps {
    fields: readonly ModelField[];
    pageId: string;
    siteId: string;
}

export function WrittenFieldsPane({ fields, pageId, siteId }: WrittenFieldsPaneProps): ReactElement {
    const [target, setTarget] = useState<ModelField | null>(null);
    const label = target === null ? "" : fieldLabel(target.key);

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {fields.length === 0 ? (
                <Note>{said.none}</Note>
            ) : (
                <ul className="flex flex-col">
                    {fields.map((held) => (
                        <li
                            key={held.key}
                            className="flex min-w-0 items-start gap-2 border-b border-inset px-3 py-1.5 last:border-b-0"
                        >
                            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                                <div className="flex min-w-0 items-center gap-2">
                                    <span className="min-w-0 truncate text-xs text-ink">{fieldLabel(held.key)}</span>
                                    <StatusBadge tone={held.ours ? "ok" : "warn"} dot={false}>
                                        {held.ours ? said.ours : said.changed}
                                    </StatusBadge>
                                </div>
                                <Link
                                    to={`/s/${siteId}/runs/${held.writtenRunId}`}
                                    title={absoluteTime(held.writtenAt)}
                                    className="w-fit font-mono text-2xs text-ink-faint hover:text-accent"
                                >
                                    {said.written(relativeTime(held.writtenAt))}
                                </Link>
                                {held.ours ? null : <p className="text-2xs text-ink-dim">{said.handBack}</p>}
                            </div>
                            <Button
                                size="sm"
                                variant="ghost"
                                icon={HistoryIcon}
                                aria-label={said.revertOne(fieldLabel(held.key))}
                                onClick={() => {
                                    setTarget(held);
                                }}
                            >
                                {said.revert}
                            </Button>
                        </li>
                    ))}
                </ul>
            )}
            <RevertFieldsDialog
                open={target !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setTarget(null);
                    }
                }}
                runId={target?.writtenRunId ?? ""}
                pageId={pageId}
                siteId={siteId}
                fields={target === null ? [] : [target.key]}
                description={target === null ? "" : target.ours ? said.ourBody(label) : said.handBackBody(label)}
            />
        </Panel>
    );
}
