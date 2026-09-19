import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import type { TabDefinition } from "../../ui/index.js";
import { AddIcon, Button, EmptyState, PublicIcon, TabPanel, Tabs } from "../../ui/index.js";
import { CreateTemplateDialog } from "./create-template.js";
import type { TemplatesQuery, TemplatesTab } from "./params.js";
import { readQuery, writeQuery } from "./params.js";
import { PolicyList } from "./policy-list.js";
import { TemplateList, useTemplateGroups } from "./template-list.js";

const tabs: readonly TabDefinition<TemplatesTab>[] = [
    { value: "templates", label: copy.templates.tabs.templates },
    { value: "policies", label: copy.templates.tabs.policies },
];

export function TemplatesScreen(): ReactElement {
    const params = useParams();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const [creating, setCreating] = useState(false);
    const groups = useTemplateGroups(siteId, query.sort);

    const change = (next: TemplatesQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    if (siteId === "") {
        return (
            <div className="flex h-full items-start justify-center p-6">
                <EmptyState icon={PublicIcon} title={copy.shell.noSiteSelected} body={copy.empty.sites} />
            </div>
        );
    }

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex shrink-0 items-start justify-between gap-3 border-b border-hairline px-3 py-2">
                <div className="flex min-w-0 flex-col gap-0.5">
                    <h1 className="text-lg font-semibold text-ink">{copy.templates.title}</h1>
                    <p className="text-xs text-ink-dim">
                        {query.tab === "templates" ? copy.templates.subtitle : copy.policies.subtitle}
                    </p>
                </div>
                {query.tab === "templates" ? (
                    <Button
                        variant="primary"
                        icon={AddIcon}
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.templates.newTemplate}
                    </Button>
                ) : null}
            </header>
            <Tabs
                className="min-h-0 flex-1"
                label={copy.templates.title}
                value={query.tab}
                onValueChange={(tab) => {
                    change({ ...query, tab, sort: null });
                }}
                tabs={tabs}
            >
                <TabPanel value="templates">
                    <TemplateList
                        siteId={siteId}
                        groups={groups}
                        sort={query.sort}
                        onSortChange={(sort) => {
                            change({ ...query, sort });
                        }}
                        onCreate={() => {
                            setCreating(true);
                        }}
                    />
                    {groups.all.length === 0 ? null : (
                        <p className="border-t border-hairline px-3 py-2 text-2xs text-ink-faint">
                            {copy.templates.siteDefaultHint}
                        </p>
                    )}
                </TabPanel>
                <TabPanel value="policies">
                    <PolicyList
                        siteId={siteId}
                        sort={query.sort}
                        onSortChange={(sort) => {
                            change({ ...query, sort });
                        }}
                    />
                </TabPanel>
            </Tabs>
            <CreateTemplateDialog
                open={creating}
                onOpenChange={setCreating}
                siteId={siteId}
                templates={groups.all}
            />
        </div>
    );
}
