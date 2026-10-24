import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import { CardBody } from "./body.js";

describe("the body a confirmation card and a resolve dialog share", () => {
    it("names the tool's family, the title, the lines and the summary", () => {
        render(
            <CardBody
                tool="findings"
                tone="warn"
                title="Resolve “Field edited on the site” on /mugs/"
                lines={<span>The SEO description was edited on the site.</span>}
                summary="Accept the site's value; nothing is written to the site"
            />,
        );

        expect(screen.getByText(copy.agent.families["findings"] ?? "")).toBeDefined();
        expect(screen.getByText("Resolve “Field edited on the site” on /mugs/")).toBeDefined();
        expect(screen.getByText("The SEO description was edited on the site.")).toBeDefined();
        expect(screen.getByText("Accept the site's value; nothing is written to the site")).toBeDefined();
    });

    it("leaves the summary out when there is none", () => {
        const { container } = render(<CardBody tool="pages_blocks" tone={null} title="Delete block 4 of /mugs/" />);

        expect(container.querySelector("p")).toBeNull();
    });
});
