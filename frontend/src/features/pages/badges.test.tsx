import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { pageStatuses } from "../../generated/vocab.js";
import { toneClasses } from "../../ui/index.js";
import { DriftBadge, PageStatusBadge, PageStatusBadges } from "./badges.js";
import { pageStatusTone } from "./labels.js";

describe("PageStatusBadge", () => {
    it.each(pageStatuses)("names a %s page in words and in its tone", (status) => {
        render(<PageStatusBadge status={status} />);
        const badge = screen.getByText(copy.pages.statuses[status]);
        expect(badge.className).toContain(toneClasses[pageStatusTone(status)].ink);
    });

    it("carries a dot unless it is told not to", () => {
        const { container, rerender } = render(<PageStatusBadge status="published" />);
        expect(container.querySelector("span[aria-hidden]")).not.toBeNull();
        rerender(<PageStatusBadge status="published" dot={false} />);
        expect(container.querySelector("span[aria-hidden]")).toBeNull();
    });

    it("shows a status this build does not know as it arrived, muted", () => {
        render(<PageStatusBadge status="teleported" />);
        expect(screen.getByText("teleported").className).toContain(toneClasses.muted.ink);
    });
});

describe("DriftBadge", () => {
    it("says drift in the warning tone with its icon", () => {
        render(<DriftBadge />);
        const badge = screen.getByText(copy.pages.drift.badge);
        expect(badge.className).toContain(toneClasses.warn.ink);
        expect(badge.querySelector("svg")).not.toBeNull();
    });
});

describe("PageStatusBadges", () => {
    it("pairs the status with the drift badge only when the page drifted", () => {
        const { rerender } = render(<PageStatusBadges page={{ status: "published", drift: false }} />);
        expect(screen.getByText(copy.pages.statuses.published)).toBeDefined();
        expect(screen.queryByText(copy.pages.drift.badge)).toBeNull();
        rerender(<PageStatusBadges page={{ status: "published", drift: true }} />);
        expect(screen.getByText(copy.pages.statuses.published)).toBeDefined();
        expect(screen.getByText(copy.pages.drift.badge)).toBeDefined();
    });
});
