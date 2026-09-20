import type { ReactElement } from "react";
import { NavLink } from "react-router";

import { copy } from "../copy/index.js";
import { CountBadge, cx } from "../ui/index.js";
import type { NavEntry } from "./nav.js";
import { railSections } from "./nav.js";

function Entry({ entry }: { entry: NavEntry }): ReactElement {
    const { to, label, Icon, badge } = entry;
    return (
        <NavLink
            to={to}
            title={label}
            className={({ isActive }) =>
                cx(
                    "relative flex h-11 w-15 flex-col items-center justify-center gap-0.5 rounded-md",
                    "transition-colors duration-100 ease-out",
                    isActive ? "bg-accent-soft text-accent" : "text-ink-dim hover:bg-inset hover:text-ink-soft",
                )
            }
        >
            {({ isActive }) => (
                <>
                    {isActive ? (
                        <span
                            aria-hidden={true}
                            className="absolute top-3 -left-2 h-5 w-0.5 rounded-full bg-accent"
                        />
                    ) : null}
                    <span className="relative">
                        <Icon size={20} />
                        {badge !== undefined && badge > 0 ? (
                            <CountBadge tone="warn" count={badge} className="absolute -top-1 -right-2.5" />
                        ) : null}
                    </span>
                    <span className="max-w-full truncate text-2xs font-medium">{label}</span>
                </>
            )}
        </NavLink>
    );
}

export interface RailProps {
    siteId: string | null;
    pending: number;
}

export function Rail({ siteId, pending }: RailProps): ReactElement {
    const sections = railSections(siteId, pending);

    return (
        <nav
            aria-label={copy.shell.sections}
            className="flex w-17 shrink-0 flex-col items-center gap-1 border-r border-hairline bg-panel py-2"
        >
            {sections.map((section, index) => (
                <div
                    key={section.key}
                    className={cx(
                        "flex w-full flex-col items-center gap-1",
                        section.pinned === true && "mt-auto",
                        index > 0 && "border-t border-hairline pt-2",
                    )}
                >
                    {section.entries.map((entry) => (
                        <Entry key={entry.key} entry={entry} />
                    ))}
                </div>
            ))}
        </nav>
    );
}
