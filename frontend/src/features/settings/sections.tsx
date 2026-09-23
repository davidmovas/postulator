import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { Banner, Button, Panel, PanelHeader, SkeletonRows } from "../../ui/index.js";
import { AdvancedCard } from "./advanced.js";
import { SettingRows } from "./rows.js";
import type { SettingSectionView, TabSettings } from "./schema.js";

const sectionTitles = copy.settings.sections as Readonly<Record<string, string>>;

export function SettingPanel({ section }: { section: SettingSectionView }): ReactElement {
    return (
        <Panel>
            <PanelHeader title={sectionTitles[section.id] ?? ""} />
            <SettingRows entries={section.entries} />
        </Panel>
    );
}

export function SettingBody({ settings }: { settings: TabSettings }): ReactElement {
    if (settings.failed) {
        return (
            <Banner
                tone="danger"
                title={copy.errors.INTERNAL}
                actions={
                    <Button size="sm" variant="secondary" onClick={settings.retry}>
                        {copy.app.retry}
                    </Button>
                }
            />
        );
    }

    if (settings.pending) {
        return (
            <Panel>
                <SkeletonRows rows={5} height={28} label={copy.settings.title} className="p-3" />
            </Panel>
        );
    }

    return (
        <>
            {settings.plain.map((section) => (
                <SettingPanel key={section.id} section={section} />
            ))}
            <AdvancedCard sections={settings.advanced} />
        </>
    );
}
