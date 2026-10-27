import { useQueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf, failure } from "../../../data/errors.js";
import { useAdoptionPreview, useAdoptParents } from "../../../data/hooks/graph.js";
import { keys } from "../../../data/keys.js";
import { pushToast } from "../../../data/toasts.js";
import type { Adoption } from "../../../data/types.js";
import {
    AccountTreeIcon,
    ArrowRightAltIcon,
    Banner,
    Button,
    cx,
    Drawer,
    EmptyState,
    SkeletonRows,
} from "../../../ui/index.js";
import { adoptionSkipLabel } from "../labels.js";

const said = copy.graph.adopt;

const drawerWidth = 640;

export interface AdoptionScope {
    pageId: string;
    path: string;
}

function names(held: Adoption["from"]): string {
    const listed = (held ?? []).map((parent) => parent.name);
    return listed.length === 0 ? said.noParent : listed.join(", ");
}

function AdoptionRow({ step }: { step: Adoption }): ReactElement {
    const skip = step.skip ?? "";
    const skipped = skip !== "";
    return (
        <li
            data-adoption={step.entityId}
            data-adoption-skip={skipped ? skip : undefined}
            className="flex flex-col gap-1 border-b border-hairline px-3 py-2 last:border-b-0"
        >
            <div className="flex min-w-0 items-baseline gap-2">
                <span className={cx("min-w-0 truncate text-xs font-semibold", skipped ? "text-ink-dim" : "text-ink")}>
                    {step.name}
                </span>
                <span className="ml-auto shrink-0 truncate font-mono text-2xs text-ink-faint">{step.path}</span>
            </div>
            <div className="flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-0.5 text-2xs">
                <span className="text-ink-faint">{said.from}</span>
                <span className={cx("min-w-0 truncate", skipped ? "text-ink-soft" : "text-ink-dim line-through")}>
                    {names(step.from)}
                </span>
                <ArrowRightAltIcon size={14} className="shrink-0 text-ink-faint" />
                <span className="text-ink-faint">{said.to}</span>
                <span className={cx("min-w-0 truncate font-semibold", skipped ? "text-ink-dim" : "text-accent")}>
                    {step.to === null ? said.top : step.to.name}
                </span>
                {step.toPath === undefined || step.toPath === "" ? null : (
                    <span className="truncate font-mono text-ink-faint">{step.toPath}</span>
                )}
            </div>
            {skipped ? (
                <span className="text-2xs text-warn">
                    {said.skipped}: {adoptionSkipLabel(skip)}
                </span>
            ) : null}
        </li>
    );
}

export interface AdoptParentsDrawerProps {
    open: boolean;
    siteId: string;
    under: AdoptionScope | null;
    onOpenChange: (open: boolean) => void;
}

export function AdoptParentsDrawer({ open, siteId, under, onOpenChange }: AdoptParentsDrawerProps): ReactElement {
    const client = useQueryClient();
    const scope = under?.pageId ?? "";
    const preview = useAdoptionPreview(siteId, scope, open);
    const apply = useAdoptParents();
    const [moved, setMoved] = useState(false);

    useEffect(() => {
        if (open) {
            setMoved(false);
            apply.reset();
        }
    }, [open, scope]);

    const pages = preview.data?.pages ?? [];
    const first = pages[0];
    const steps = useMemo(() => pages.flatMap((page) => page.steps.items ?? []), [pages]);
    const adopting = first?.adopting ?? 0;
    const skipped = first?.skipped ?? 0;
    const hash = first?.hash ?? "";
    const refusal = apply.error === null || moved ? null : errorMessageOf(apply.error);

    const write = (): void => {
        setMoved(false);
        apply.mutate(
            { siteId, under: scope, hash },
            {
                onSuccess: () => {
                    pushToast("info", said.applied(adopting));
                    onOpenChange(false);
                },
                onError: (thrown) => {
                    if (failure(thrown).code === "CONFLICT") {
                        setMoved(true);
                        void client.resetQueries({ queryKey: keys.graph.adoption(siteId, scope) });
                    }
                },
            },
        );
    };

    const body = ((): ReactElement => {
        if (preview.isPending) {
            return <SkeletonRows rows={6} label={said.loading} />;
        }
        if (preview.error !== null && first === undefined) {
            return <Banner tone="danger" title={errorMessageOf(preview.error) ?? said.loading} />;
        }
        if (steps.length === 0) {
            return <EmptyState icon={AccountTreeIcon} title={said.none} body={said.noneBody} />;
        }
        return (
            <>
                <p className="text-xs font-medium text-ink" data-adoption-counts={true}>
                    {said.counts(adopting, skipped)}
                </p>
                <ul className="flex flex-col overflow-hidden rounded-md border border-hairline bg-panel">
                    {steps.map((step) => (
                        <AdoptionRow key={step.entityId} step={step} />
                    ))}
                </ul>
                {preview.hasNextPage ? (
                    <Button
                        size="sm"
                        variant="ghost"
                        className="self-start"
                        busy={preview.isFetchingNextPage}
                        onClick={() => {
                            void preview.fetchNextPage();
                        }}
                    >
                        {said.more}
                    </Button>
                ) : null}
            </>
        );
    })();

    return (
        <Drawer
            open={open}
            onOpenChange={onOpenChange}
            title={said.title}
            description={said.body}
            closeLabel={said.cancel}
            width={drawerWidth}
            footer={
                <>
                    <Button
                        variant="ghost"
                        onClick={() => {
                            onOpenChange(false);
                        }}
                    >
                        {said.cancel}
                    </Button>
                    <Button
                        variant="primary"
                        icon={AccountTreeIcon}
                        data-adoption-apply={true}
                        disabled={adopting === 0 || hash === "" || preview.isFetching}
                        busy={apply.isPending}
                        onClick={write}
                    >
                        {apply.isPending ? said.applying : said.apply(adopting)}
                    </Button>
                </>
            }
        >
            <div className="flex flex-col gap-3 p-3">
                <p className="flex min-w-0 items-center gap-1.5 text-xs text-ink-soft">
                    <AccountTreeIcon size={14} className="shrink-0 text-ink-faint" />
                    <span className="truncate">{under === null ? said.whole : said.under(under.path)}</span>
                </p>
                {moved ? <Banner tone="warn" title={said.moved} /> : null}
                {refusal === null ? null : <Banner tone="danger" title={refusal} />}
                {body}
            </div>
        </Drawer>
    );
}
