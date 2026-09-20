import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useSite, useUpdateSite } from "../../data/hooks/sites.js";
import { useDeletePolicy } from "../../data/hooks/templates.js";
import type { LinkPolicy } from "../../data/types.js";
import { AddIcon, Banner, Button, CountBadge, Screen } from "../../ui/index.js";
import { tabPath, TemplateTabs } from "./header.js";
import { wantsNew } from "./params.js";
import { PolicyDetail } from "./policy-detail.js";
import { PolicyDrawer } from "./policy-form.js";
import { usePolicyRows } from "./policy-rows.js";
import { PolicyTable } from "./policy-table.js";

export function PoliciesScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const rows = usePolicyRows(siteId);
    const site = useSite(siteId === "" ? null : siteId);
    const updateSite = useUpdateSite();
    const remove = useDeletePolicy();
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [editing, setEditing] = useState<LinkPolicy | null>(null);
    const [creating, setCreating] = useState(false);
    const asked = wantsNew(searchParams);

    useEffect(() => {
        if (asked) {
            setCreating(true);
            setSearchParams(new URLSearchParams(), { replace: true });
        }
    }, [asked, setSearchParams]);

    const selected = rows.shown.find((held) => held.id === selectedId) ?? rows.shown[0] ?? null;
    const chosenId = site.data?.site.defaults.linkPolicyId ?? null;
    const failed = rows.error !== null && rows.error !== undefined;
    const seed = rows.shown.find((held) => held.id === rows.inForceId) ?? rows.shown[0] ?? null;

    const useHere = (): void => {
        const held = site.data?.site;
        if (held === undefined || selected === null) {
            return;
        }
        updateSite.mutate({ id: held.id, defaults: { ...held.defaults, linkPolicyId: selected.id } });
    };

    return (
        <Screen
            title={copy.templates.title}
            badge={rows.pending ? undefined : <CountBadge tone="muted" count={rows.shown.length} />}
            variant="split"
            tabs={<TemplateTabs siteId={siteId} tab="policies" />}
            actions={
                <>
                    <Button
                        onClick={() => {
                            void navigate(`${tabPath(siteId, "templates")}?action=new`);
                        }}
                    >
                        {copy.templates.newTemplate}
                    </Button>
                    <Button
                        variant="primary"
                        icon={AddIcon}
                        data-policy-new={true}
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.policies.newPolicy}
                    </Button>
                </>
            }
            right={
                selected === null || rows.pending ? null : (
                    <PolicyDetail
                        key={selected.id}
                        policy={selected}
                        inForce={selected.id === rows.inForceId}
                        chosenHere={selected.id === chosenId}
                        applying={updateSite.isPending}
                        deleting={remove.isPending}
                        onEdit={() => {
                            setEditing(selected);
                        }}
                        onUseHere={useHere}
                        onDelete={() => {
                            remove.mutate(
                                { id: selected.id },
                                {
                                    onSuccess: () => {
                                        setSelectedId(null);
                                    },
                                },
                            );
                        }}
                    />
                )
            }
        >
            {failed ? (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={failure(rows.error).message}
                        actions={<Button onClick={rows.retry}>{copy.app.retry}</Button>}
                    />
                </div>
            ) : (
                <PolicyTable
                    rows={rows.shown}
                    pending={rows.pending}
                    selectedId={selected?.id ?? null}
                    inForceId={rows.inForceId}
                    hasMore={rows.hasMore}
                    loadingMore={rows.loadingMore}
                    onSelect={setSelectedId}
                    onLoadMore={rows.loadMore}
                    onCreate={() => {
                        setCreating(true);
                    }}
                />
            )}
            {editing === null && !creating ? null : (
                <PolicyDrawer
                    siteId={siteId}
                    policy={editing}
                    seed={seed}
                    onClose={() => {
                        setEditing(null);
                        setCreating(false);
                    }}
                />
            )}
        </Screen>
    );
}
