import type { ReactElement, ReactNode } from "react";
import { useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { detailCodeOf, errorMessageOf, failure } from "../../data/errors.js";
import { useClearSiteData, useSiteDataHeld } from "../../data/hooks/sites.js";
import type { Site, SiteDataHeld } from "../../data/types.js";
import { Banner, Button, Checkbox, Dialog, RestartAltIcon, Skeleton } from "../../ui/index.js";

const said = copy.sites.clear;

export const siteBusyCode = "site_busy";

export interface ClearChoice {
    model: boolean;
    graph: boolean;
    history: boolean;
}

export function chosen(choice: ClearChoice): ClearChoice {
    return { ...choice, history: choice.history || choice.model };
}

export function nothingChosen(choice: ClearChoice): boolean {
    return !choice.model && !choice.graph && !choice.history;
}

export function isSiteBusy(thrown: unknown): boolean {
    if (thrown === null || thrown === undefined) {
        return false;
    }
    return detailCodeOf(failure(thrown)) === siteBusyCode;
}

export function quietProblem(thrown: unknown, quiet: readonly string[]): string | null {
    if (thrown === null || thrown === undefined || isSiteBusy(thrown)) {
        return null;
    }
    const reported = failure(thrown);
    if (!quiet.includes(reported.code)) {
        return null;
    }
    return reported.message === "" ? errorMessageOf(thrown) : reported.message;
}

interface GroupProps {
    name: string;
    label: string;
    checked: boolean;
    disabled?: boolean;
    onChange: (next: boolean) => void;
    children: ReactNode;
}

function Group({ name, label, checked, disabled = false, onChange, children }: GroupProps): ReactElement {
    return (
        <div className="flex flex-col gap-1 border-b border-inset px-3 py-2.5 last:border-b-0" data-clear-group={name}>
            <Checkbox
                label={label}
                checked={checked}
                disabled={disabled}
                onChange={(event) => {
                    onChange(event.target.checked);
                }}
            />
            <div className="pl-6 text-xs leading-snug text-ink-soft">{children}</div>
        </div>
    );
}

function Hint({ held, children }: { held: SiteDataHeld | undefined; children: (held: SiteDataHeld) => ReactNode }): ReactElement {
    return held === undefined ? <Skeleton width="80%" height={12} /> : <>{children(held)}</>;
}

export function BusyBanner({ siteId }: { siteId: string }): ReactElement {
    return (
        <Banner
            tone="warn"
            title={copy.sites.busy}
            actions={
                <Link to={`/s/${siteId}/runs`}>
                    <Button variant="secondary" size="sm">
                        {copy.sites.openRuns}
                    </Button>
                </Link>
            }
        />
    );
}

export interface ClearDataDialogProps {
    site: Site;
    onClose: () => void;
}

export function ClearDataDialog({ site, onClose }: ClearDataDialogProps): ReactElement {
    const [choice, setChoice] = useState<ClearChoice>({ model: false, graph: false, history: false });
    const counts = useSiteDataHeld(site.id, true);
    const clear = useClearSiteData();
    const held = counts.data?.held;
    const busy = (counts.data?.busy ?? []).length > 0 || isSiteBusy(clear.error);
    const asked = chosen(choice);
    const refusal = counts.isPending ? said.counting : nothingChosen(asked) ? said.nothing : busy ? copy.sites.busy : null;
    const problem = quietProblem(clear.error, ["CONFLICT", "INVALID"]);

    return (
        <Dialog
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={said.title}
            description={said.body(site.name)}
            icon={RestartAltIcon}
            destructive={true}
            confirmLabel={said.confirm}
            cancelLabel={copy.app.cancel}
            busy={clear.isPending}
            refusal={refusal}
            onConfirm={() => {
                if (refusal !== null) {
                    return;
                }
                clear.mutate({ id: site.id, ...asked }, { onSuccess: onClose });
            }}
        >
            <div className="overflow-hidden rounded-lg border border-hairline bg-panel" data-clear-groups={true}>
                <Group
                    name="model"
                    label={said.model}
                    checked={choice.model}
                    onChange={(next) => {
                        setChoice((held) => ({ ...held, model: next }));
                    }}
                >
                    <Hint held={held}>{(known) => said.modelHint(known.pages, known.findings)}</Hint>
                </Group>
                <Group
                    name="graph"
                    label={said.graph}
                    checked={choice.graph}
                    onChange={(next) => {
                        setChoice((held) => ({ ...held, graph: next }));
                    }}
                >
                    <Hint held={held}>{(known) => said.graphHint(known.entities, known.edges)}</Hint>
                </Group>
                <Group
                    name="history"
                    label={said.history}
                    checked={asked.history}
                    disabled={choice.model}
                    onChange={(next) => {
                        setChoice((held) => ({ ...held, history: next }));
                    }}
                >
                    <Hint held={held}>
                        {(known) => (
                            <>
                                {said.historyHint(known.runs, known.chats)}
                                {known.unfinished > 0 ? ` ${said.unfinished(known.unfinished)}` : null}
                                {choice.model ? <span className="mt-1 block text-ink-faint">{said.historyTaken}</span> : null}
                            </>
                        )}
                    </Hint>
                </Group>
            </div>
            {busy ? <BusyBanner siteId={site.id} /> : null}
            {problem === null ? null : <Banner tone="danger" title={problem} />}
        </Dialog>
    );
}
