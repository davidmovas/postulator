import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { renderScreen } from "../../../testing/render.js";
import { ResearchPane } from "./research.js";

const said = copy.runs.review.research;

describe("the research pane", () => {
    it("lists the facts and names each source by its title and its site", () => {
        renderScreen(
            <ResearchPane
                payload={{
                    text: "- Caffeine peaks within an hour [who.int]",
                    sources: [{ url: "https://www.who.int/caffeine", title: "Caffeine facts" }],
                    searches: 2,
                }}
            />,
        );

        expect(screen.getByText("Caffeine peaks within an hour [who.int]")).toBeDefined();
        expect(screen.getByText(said.summary(1, 1, 2))).toBeDefined();
        const source = screen.getByRole("button", { name: /Caffeine facts/ });
        expect(source.getAttribute("title")).toBe("https://www.who.int/caffeine");
        expect(screen.getByText("www.who.int")).toBeDefined();
    });

    it("says so when the search confirmed nothing", () => {
        renderScreen(<ResearchPane payload={{ text: "", sources: [], searches: 1 }} />);

        expect(screen.getByText(said.noFacts)).toBeDefined();
        expect(screen.getByText(said.noSources)).toBeDefined();
    });
});
