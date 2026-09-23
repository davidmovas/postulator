import type { ReactElement } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { Tabs } from "../../ui/index.js";
import type { TabItem } from "../../ui/index.js";
import type { TemplatesTab } from "./params.js";

const tabs: readonly TabItem<TemplatesTab>[] = [
    { key: "templates", label: copy.templates.tabs.templates },
    { key: "policies", label: copy.templates.tabs.policies },
];

export function tabPath(siteId: string, tab: TemplatesTab): string {
    return tab === "policies" ? `/s/${siteId}/templates/policies` : `/s/${siteId}/templates`;
}

export interface TemplateTabsProps {
    siteId: string;
    tab: TemplatesTab;
}

export function TemplateTabs({ siteId, tab }: TemplateTabsProps): ReactElement {
    const navigate = useNavigate();
    return (
        <Tabs
            label={copy.templates.title}
            items={tabs}
            value={tab}
            onValueChange={(next) => {
                void navigate(tabPath(siteId, next));
            }}
        />
    );
}
