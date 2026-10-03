import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useDeletePage, usePage } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { pageCategoryItems } from "../../domain/categories.js";
import type { TabItem } from "../../ui/index.js";
import {
    Button,
    CategoryTrail,
    Checkbox,
    DeleteIcon,
    Dialog,
    Drawer,
    SkeletonRows,
    SmartToyIcon,
    TabPanel,
    Tabs,
    WarningIcon,
} from "../../ui/index.js";
import { askAgent } from "../agent/index.js";
import { PageStatusBadges } from "./badges.js";
import type { PageTab } from "./params.js";
import { PageDetails } from "./details.js";
import type { EntityIndex } from "./entities.js";
import { PageLinks } from "./links.js";
import { PageMapping } from "./mapping.js";
import { PreviewTab } from "./preview/tab.js";
import { PageReportPanel } from "./report.js";
import { TemplatePanel } from "./template.js";

const drawerWidth = 688;

const tabs: readonly TabItem<PageTab>[] = [
    { key: "details", label: copy.pages.tabs.details },
    { key: "links", label: copy.pages.tabs.links },
    { key: "mapping", label: copy.pages.tabs.mapping },
    { key: "report", label: copy.pages.tabs.report },
    { key: "preview", label: copy.pages.tabs.preview },
];

interface FiledUnderProps {
    page: Pick<Page, "categories" | "categoriesNeedPlugin">;
}

function FiledUnder({ page }: FiledUnderProps): ReactElement | null {
    const filed = pageCategoryItems(page);
    if (filed.length === 0) {
        return null;
    }
    return (
        <div data-page-categories={true} className="flex shrink-0 flex-col gap-1 border-b border-hairline px-3 py-1.5">
            <div className="flex min-w-0 items-center gap-2">
                <span className="shrink-0 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                    {copy.pages.detail.filedUnder}
                </span>
                <CategoryTrail items={filed} label={copy.categories.trail} />
            </div>
            {page.categoriesNeedPlugin ? (
                <p data-categories-need-plugin={true} className="flex items-start gap-1.5 text-xs text-warn">
                    <WarningIcon size={14} className="mt-px shrink-0" />
                    <span>{copy.pages.summary.categoriesNeedPlugin}</span>
                </p>
            ) : null}
        </div>
    );
}

export interface PageDrawerProps {
    pageId: string;
    siteId: string;
    index: EntityIndex;
    search: string;
    tab: PageTab;
    onTabChange: (tab: PageTab) => void;
    onClose: () => void;
}

export function PageDrawer({
    pageId,
    siteId,
    index,
    search,
    tab: active,
    onTabChange: setActive,
    onClose,
}: PageDrawerProps): ReactElement {
    const detail = usePage(pageId);
    const remove = useDeletePage();
    const [confirming, setConfirming] = useState(false);
    const [onSite, setOnSite] = useState(false);
    const page = detail.data?.page;
    const live = page?.wpId !== null && page?.wpId !== undefined && page.wpId !== 0;

    return (
        <Drawer
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={page?.path ?? copy.app.loading}
            closeLabel={copy.pages.detail.close}
            width={drawerWidth}
            header={
                page === undefined ? null : (
                    <div className="flex shrink-0 items-center gap-1.5">
                        <PageStatusBadges page={page} />
                    </div>
                )
            }
            footer={
                page === undefined ? null : (
                    <>
                        <Button
                            className="mr-auto"
                            variant="danger"
                            icon={DeleteIcon}
                            onClick={() => {
                                setConfirming(true);
                            }}
                        >
                            {copy.pages.detail.deletePage}
                        </Button>
                        <Button
                            variant="ghost"
                            icon={SmartToyIcon}
                            onClick={() => {
                                askAgent(copy.agent.ask.page(page.path, page.id));
                            }}
                        >
                            {copy.agent.askAbout}
                        </Button>
                        <Button onClick={onClose}>{copy.pages.detail.close}</Button>
                    </>
                )
            }
        >
            {detail.isPending ? (
                <div className="p-3">
                    <SkeletonRows rows={8} label={copy.app.loading} />
                </div>
            ) : page === undefined ? (
                <div className="p-3">
                    <p className="text-sm text-ink">
                        {failure(detail.error).code === "NOT_FOUND"
                            ? copy.pages.detail.notFound
                            : failure(detail.error).message}
                    </p>
                </div>
            ) : (
                <div className="flex h-full min-h-0 flex-col">
                    <FiledUnder page={page} />
                    <div className="flex h-8 shrink-0 items-center border-b border-hairline px-2">
                        <Tabs
                            label={copy.pages.title}
                            items={tabs}
                            value={active}
                            onValueChange={setActive}
                        />
                    </div>
                    <TabPanel label={copy.pages.tabs.details} active={active === "details"}>
                        <PageDetails page={page} siteId={siteId} search={search} />
                    </TabPanel>
                    <TabPanel label={copy.pages.tabs.links} active={active === "links"}>
                        <div className="p-3">
                            <PageLinks links={detail.data?.links ?? []} siteId={siteId} search={search} />
                        </div>
                    </TabPanel>
                    <TabPanel label={copy.pages.tabs.mapping} active={active === "mapping"}>
                        <div className="flex flex-col gap-3 p-3">
                            <PageMapping page={page} siteId={siteId} index={index} search={search} />
                            <TemplatePanel page={page} siteId={siteId} />
                        </div>
                    </TabPanel>
                    <TabPanel label={copy.pages.tabs.report} active={active === "report"}>
                        <div className="p-3">
                            <PageReportPanel pageId={page.id} siteId={siteId} />
                        </div>
                    </TabPanel>
                    <TabPanel label={copy.pages.tabs.preview} active={active === "preview"}>
                        <PreviewTab page={page} siteId={siteId} />
                    </TabPanel>
                    <Dialog
                        open={confirming}
                        onOpenChange={(next) => {
                            setConfirming(next);
                            if (!next) {
                                setOnSite(false);
                            }
                        }}
                        title={copy.pages.detail.deleteTitle}
                        description={copy.pages.detail.deleteBody}
                        confirmLabel={copy.pages.detail.deleteConfirm}
                        cancelLabel={copy.pages.detail.cancel}
                        destructive={true}
                        icon={DeleteIcon}
                        busy={remove.isPending}
                        onConfirm={() => {
                            remove.mutate(
                                { id: page.id, onSite: onSite && live },
                                {
                                    onSuccess: () => {
                                        setConfirming(false);
                                        setOnSite(false);
                                        onClose();
                                    },
                                },
                            );
                        }}
                    >
                        <div className="flex flex-col gap-1.5 rounded-md border border-hairline bg-panel px-3 py-2.5">
                            <Checkbox
                                data-page-delete-on-site={true}
                                label={copy.pages.detail.deleteOnSite}
                                checked={onSite && live}
                                disabled={!live}
                                onChange={(event) => {
                                    setOnSite(event.currentTarget.checked);
                                }}
                            />
                            <p className="text-2xs text-ink-faint">
                                {live
                                    ? copy.pages.detail.deleteOnSiteHint
                                    : copy.pages.detail.deleteOnSiteUnavailable}
                            </p>
                        </div>
                    </Dialog>
                </div>
            )}
        </Drawer>
    );
}
