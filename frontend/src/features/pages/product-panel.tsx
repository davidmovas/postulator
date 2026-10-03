import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { usePageReport } from "../../data/hooks/reports.js";
import type { Page } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { Banner, CategoryIcon } from "../../ui/index.js";
import { nameDiffers, productOutputsOf } from "./product.js";

const said = copy.pages.detail.product;

export interface ProductPanelProps {
    page: Page;
}

export function ProductPanel({ page }: ProductPanelProps): ReactElement {
    const report = usePageReport(page.wpId === null ? null : page.id);

    if (page.wpId === null) {
        return <Banner tone="warn" icon={CategoryIcon} title={said.notInStore} body={said.notInStoreBody} />;
    }

    const storeName = page.observed.title;
    const outputs = productOutputsOf(report.data?.product ?? null);
    const finished = report.data?.finishedAt ?? null;

    return (
        <section aria-label={said.title} className="flex flex-col gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2">
            <span className="text-xs font-medium text-ink-dim">{said.title}</span>
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                <dt className="text-ink-faint">{said.address}</dt>
                <dd className="min-w-0 truncate font-mono text-ink-soft" title={page.path}>
                    {page.path}
                </dd>
                {page.plannedPath === "" ? null : (
                    <>
                        <dt className="text-ink-faint">{said.fromSheet}</dt>
                        <dd className="min-w-0 truncate font-mono text-ink-soft" title={page.plannedPath}>
                            {page.plannedPath}
                        </dd>
                    </>
                )}
                {storeName === "" ? null : (
                    <>
                        <dt className="text-ink-faint">{said.name}</dt>
                        <dd className="min-w-0 text-ink-soft">{storeName}</dd>
                    </>
                )}
            </dl>
            {nameDiffers(page.h1, storeName) ? <p className="text-2xs text-warn">{said.nameDiffers(page.h1)}</p> : null}
            {outputs === null ? (
                <p className="text-2xs text-ink-faint">{said.nothingWritten}</p>
            ) : (
                <div className="flex flex-col gap-1.5">
                    {finished === null ? null : (
                        <span className="text-2xs text-ink-faint" title={absoluteTime(finished)}>
                            {said.lastWritten(relativeTime(finished))}
                        </span>
                    )}
                    {outputs.shortDescription === "" ? null : (
                        <div className="flex flex-col gap-0.5">
                            <span className="text-2xs text-ink-faint">{said.short}</span>
                            <p className="text-xs text-ink-soft">{outputs.shortDescription}</p>
                        </div>
                    )}
                    {outputs.specifications.length === 0 ? null : (
                        <div className="flex flex-col gap-0.5">
                            <span className="text-2xs text-ink-faint">{said.attributes}</span>
                            <dl className="grid grid-cols-[auto_1fr] gap-x-3 text-xs">
                                {outputs.specifications.map((row) => (
                                    <div key={row.name} className="contents">
                                        <dt className="text-ink-faint">{row.name}</dt>
                                        <dd className="text-ink-soft">{row.value}</dd>
                                    </div>
                                ))}
                            </dl>
                        </div>
                    )}
                </div>
            )}
            <p className="text-2xs text-ink-faint">{said.untouched}</p>
        </section>
    );
}
