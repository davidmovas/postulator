import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { formErrorOf } from "../../data/errors.js";
import { useBrowserLocation, useSetTorPath, torPathKey } from "../../data/hooks/browser.js";
import { openExternal, pickOpenFile } from "../../data/host.js";
import {
    Banner,
    Button,
    Panel,
    PanelHeader,
    PublicIcon,
    Skeleton,
    StatusBadge,
    TravelExploreIcon,
} from "../../ui/index.js";
import { AdvancedCard } from "./advanced.js";
import { useTabSettings } from "./schema.js";

const said = copy.settings.browser;

const chosenBySetting = "setting";

export function BrowserSettingsScreen(): ReactElement {
    const located = useBrowserLocation();
    const settings = useTabSettings("browser");
    const write = useSetTorPath();

    const entry = settings.plain
        .flatMap((section) => section.entries)
        .find((held) => held.descriptor.key === torPathKey);
    const configured = typeof entry?.value === "string" ? entry.value : "";

    const answer = located.data;
    const installed = answer?.installed === true;
    const rejected = configured !== "" && answer !== undefined && answer.source !== chosenBySetting;
    const refused = formErrorOf(write.error);

    const choose = (): void => {
        void pickOpenFile({
            title: said.chooseTitle,
            filters: [{ displayName: said.filter, pattern: "firefox.exe" }],
        }).then((picked) => {
            if (picked !== null) {
                write.mutate(picked);
            }
        });
    };

    return (
        <div className="flex max-w-2xl flex-col gap-4">
            <Panel>
                <PanelHeader title={said.title}>
                    {located.isPending ? null : (
                        <StatusBadge tone={installed ? "ok" : "warn"} icon={installed ? PublicIcon : undefined}>
                            {installed ? said.found : said.missing}
                        </StatusBadge>
                    )}
                </PanelHeader>
                <div className="flex flex-col gap-3 p-3">
                    {located.isPending ? (
                        <Skeleton height={28} />
                    ) : installed ? (
                        <div className="flex flex-col gap-1">
                            <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">
                                {said.startsFrom}
                            </span>
                            <div className="flex items-center gap-2">
                                <div
                                    title={answer?.path}
                                    className="flex h-7 min-w-0 flex-1 items-center truncate rounded-md border border-hairline bg-inset px-2 font-mono text-xs text-ink-soft"
                                >
                                    {answer?.path}
                                </div>
                                <StatusBadge tone="muted" dot={false}>
                                    {answer?.source === chosenBySetting ? said.sourceChosen : said.sourceDetected}
                                </StatusBadge>
                            </div>
                        </div>
                    ) : (
                        <p className="text-xs text-ink-dim">{said.missingBody}</p>
                    )}

                    {rejected ? <Banner tone="danger" title={said.notTor} /> : null}
                    {refused === null ? null : <Banner tone="danger" title={refused} />}

                    <div className="flex items-center gap-2">
                        <Button
                            variant={installed ? "secondary" : "primary"}
                            busy={write.isPending}
                            onClick={choose}
                        >
                            {said.choose}
                        </Button>
                        {configured === "" ? null : (
                            <Button
                                variant="ghost"
                                title={said.clearHint}
                                busy={write.isPending}
                                onClick={() => {
                                    write.mutate("");
                                }}
                            >
                                {said.clear}
                            </Button>
                        )}
                        <div className="ml-auto">
                            <Button
                                variant="secondary"
                                icon={TravelExploreIcon}
                                disabled={!installed}
                                title={installed ? said.testHint : said.testBlocked}
                                onClick={() => {
                                    void openExternal(said.testUrl);
                                }}
                            >
                                {said.test}
                            </Button>
                        </div>
                    </div>
                </div>
            </Panel>
            <AdvancedCard sections={settings.advanced} />
        </div>
    );
}
