import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { KeyboardArrowDownIcon, KeyboardArrowUpIcon, Panel, SectionLabel } from "../../ui/index.js";
import { SettingRows } from "./rows.js";
import type { SettingSectionView } from "./schema.js";

const sectionTitles = copy.settings.sections as Readonly<Record<string, string>>;

export function AdvancedCard({ sections }: { sections: readonly SettingSectionView[] }): ReactElement | null {
    const [open, setOpen] = useState(false);

    if (sections.length === 0) {
        return null;
    }

    const count = sections.reduce((total, section) => total + section.entries.length, 0);
    const Chevron = open ? KeyboardArrowUpIcon : KeyboardArrowDownIcon;

    return (
        <Panel>
            <button
                type="button"
                aria-expanded={open}
                className="flex h-8 shrink-0 items-center gap-2 px-3 text-left transition-colors duration-100 ease-out hover:bg-inset"
                onClick={() => {
                    setOpen(!open);
                }}
            >
                <Chevron size={14} className="text-ink-faint" />
                <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">
                    {copy.settings.advanced}
                </span>
                <span className="text-2xs text-ink-faint">{copy.settings.advancedCount(count)}</span>
            </button>
            {open
                ? sections.map((section) => (
                      <div key={section.id} className="border-t border-hairline">
                          <SectionLabel className="px-3 pt-2 pb-1">
                              {sectionTitles[section.id] ?? copy.settings.advanced}
                          </SectionLabel>
                          <SettingRows entries={section.entries} />
                      </div>
                  ))
                : null}
        </Panel>
    );
}
