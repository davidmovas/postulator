import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { isBrowsable, openExternal } from "../../../data/host.js";
import type { PageModel } from "../../../data/types.js";
import { absoluteTime, relativeTime } from "../../../domain/format.js";
import { OpenInNewIcon, Panel, PanelHeader } from "../../../ui/index.js";
import { Fact, Facts, Note } from "./facts.js";
import { taxonomyLabel, wpStatusLabel, wpTypeLabel } from "./labels.js";

const said = copy.pages.model;

export interface IdentityPaneProps {
    model: PageModel;
    onSite: boolean;
}

export function IdentityPane({ model, onSite }: IdentityPaneProps): ReactElement {
    const identity = model.identity;
    const terms = model.terms ?? [];

    return (
        <Panel>
            <PanelHeader title={said.identity.title} />
            {identity === null ? (
                <Note>{onSite ? said.notObserved : said.notOnSite}</Note>
            ) : (
                <Facts>
                    <Fact label={said.identity.address} title={identity.url}>
                        {isBrowsable(identity.url) ? (
                            <button
                                type="button"
                                title={copy.app.openExternal}
                                onClick={() => {
                                    void openExternal(identity.url);
                                }}
                                className="flex min-w-0 max-w-full items-center gap-1 text-left font-mono text-accent hover:underline"
                            >
                                <span className="truncate">{identity.url}</span>
                                <OpenInNewIcon size={12} className="shrink-0" />
                            </button>
                        ) : (
                            <span className="font-mono">{identity.url}</span>
                        )}
                    </Fact>
                    <Fact label={said.identity.type}>
                        {model.wpId === null
                            ? wpTypeLabel(identity.wpType)
                            : said.identity.typeWithId(wpTypeLabel(identity.wpType), model.wpId)}
                    </Fact>
                    <Fact label={said.identity.status}>{wpStatusLabel(identity.status)}</Fact>
                    <Fact label={said.identity.modified} title={absoluteTime(identity.modifiedGmt)}>
                        {relativeTime(identity.modifiedGmt)}
                    </Fact>
                    <Fact label={said.identity.observed} title={absoluteTime(identity.observedAt)}>
                        {relativeTime(identity.observedAt)}
                    </Fact>
                    {terms.length === 0 ? null : (
                        <Fact label={said.identity.terms}>
                            <span className="flex flex-wrap gap-x-2 gap-y-0.5 whitespace-normal">
                                {terms.map((term) => (
                                    <span key={`${term.taxonomy}:${term.slug}`} title={taxonomyLabel(term.taxonomy)}>
                                        {term.name}
                                    </span>
                                ))}
                            </span>
                        </Fact>
                    )}
                </Facts>
            )}
        </Panel>
    );
}
