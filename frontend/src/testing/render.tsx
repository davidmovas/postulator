import type { ReactElement, ReactNode } from "react";
import { render } from "@testing-library/react";
import type { RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router";

export function renderScreen(node: ReactNode, at = "/"): RenderResult {
    const wrapped = (<MemoryRouter initialEntries={[at]}>{node}</MemoryRouter>) as ReactElement;
    return render(wrapped);
}
