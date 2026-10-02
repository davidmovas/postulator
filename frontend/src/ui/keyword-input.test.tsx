import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { Keyword } from "../data/types.js";
import { KeywordInput } from "./keyword-input.js";

const labels = { removeLabel: "Remove", phraseLabel: "Keyword", volumeLabel: "Volume" };

function draw(values: Keyword[], onChange: (values: Keyword[]) => void = vi.fn()): void {
    render(<KeywordInput values={values} onChange={onChange} {...labels} />);
}

describe("KeywordInput", () => {
    it("shows each keyword with its volume, the most searched first", () => {
        draw([{ text: "bpc-157" }, { text: "bpc 157", volume: 12000 }]);

        const chips = screen.getAllByRole("listitem");
        expect(chips.map((chip) => chip.textContent)).toStrictEqual(["bpc 15712,000", "bpc-157"]);
    });

    it("adds a phrase with a volume and keeps the list ordered", () => {
        const onChange = vi.fn();
        draw([{ text: "bpc-157" }, { text: "bpc 157", volume: 12000 }], onChange);

        fireEvent.change(screen.getByLabelText("Keyword"), { target: { value: " buy bpc 157 " } });
        fireEvent.change(screen.getByLabelText("Volume"), { target: { value: "5400" } });
        fireEvent.keyDown(screen.getByLabelText("Keyword"), { key: "Enter" });

        expect(onChange).toHaveBeenCalledWith([
            { text: "bpc 157", volume: 12000 },
            { text: "buy bpc 157", volume: 5400 },
            { text: "bpc-157" },
        ]);
    });

    it("adds a phrase without a volume", () => {
        const onChange = vi.fn();
        draw([], onChange);

        fireEvent.change(screen.getByLabelText("Keyword"), { target: { value: "tb 500" } });
        fireEvent.keyDown(screen.getByLabelText("Volume"), { key: "Enter" });

        expect(onChange).toHaveBeenCalledWith([{ text: "tb 500" }]);
    });

    it("gives a phrase it already holds the volume typed for it", () => {
        const onChange = vi.fn();
        draw([{ text: "bpc 157" }], onChange);

        fireEvent.change(screen.getByLabelText("Keyword"), { target: { value: "BPC 157" } });
        fireEvent.change(screen.getByLabelText("Volume"), { target: { value: "300" } });
        fireEvent.keyDown(screen.getByLabelText("Keyword"), { key: "Enter" });

        expect(onChange).toHaveBeenCalledWith([{ text: "bpc 157", volume: 300 }]);
    });

    it("adds nothing for a blank phrase", () => {
        const onChange = vi.fn();
        draw([{ text: "bpc 157" }], onChange);

        fireEvent.change(screen.getByLabelText("Volume"), { target: { value: "300" } });
        fireEvent.keyDown(screen.getByLabelText("Keyword"), { key: "Enter" });

        expect(onChange).not.toHaveBeenCalled();
    });

    it("removes a keyword", () => {
        const onChange = vi.fn();
        draw([{ text: "bpc 157", volume: 12000 }, { text: "bpc-157" }], onChange);

        fireEvent.click(screen.getByRole("button", { name: "Remove bpc 157" }));

        expect(onChange).toHaveBeenCalledWith([{ text: "bpc-157" }]);
    });
});
