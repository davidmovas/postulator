import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { syncStages, topicSkips } from "../../generated/vocab.js";
import { aProgress } from "../../testing/sync.js";
import {
    leftReasons,
    pageCount,
    stagePhrase,
    stageSteps,
    stageWords,
    stoppedTitle,
    syncConflictOf,
    syncLine,
    syncRetrying,
    syncRunPath,
} from "./model.js";

const said = copy.sync;

function refused(cause: unknown): Error {
    return Object.assign(new Error("refused"), { cause });
}

describe("a sync's stage in words", () => {
    it("has words for every stage the walk goes through", () => {
        for (const stage of syncStages) {
            expect(stageWords(stage), stage).not.toBe("");
            expect(stageWords(stage), stage).not.toBe(stage);
        }
    });

    it("says the walk has not started before it reaches a stage", () => {
        expect(stageWords("")).toBe(said.waiting);
    });

    it.each([
        ["inventory", "Reading the site's setup"],
        ["content", "Reading the pages"],
        ["store", "Reading the store"],
        ["render", "Visiting the pages"],
        ["derive", "Deriving the page tree"],
        ["topics", "Building the topical map"],
        ["fields", "Reading custom fields"],
        ["done", "Done"],
    ])("names %s as %s", (stage, words) => {
        expect(stageWords(stage)).toBe(words);
    });
});

describe("why the topical map left pages without an entity", () => {
    it("has words for every reason the map can give", () => {
        for (const reason of topicSkips) {
            const words = leftReasons(aProgress({ unplacedBy: { [reason]: 1 } }));
            expect(
                words.map((left) => left.words),
                reason,
            ).toEqual([said.left.reasons[reason]]);
            expect(words[0]?.words, reason).not.toBe(reason);
        }
    });

    it("lists the commonest reason first and the rest by their words", () => {
        const left = leftReasons(aProgress({ unplacedBy: { removed: 2, disagrees: 2, unknown_link: 5 } }));
        expect(left).toEqual([
            { reason: "unknown_link", words: said.left.reasons.unknown_link, count: 5 },
            { reason: "removed", words: said.left.reasons.removed, count: 2 },
            { reason: "disagrees", words: said.left.reasons.disagrees, count: 2 },
        ]);
    });

    it("lists nothing when no reasons came back", () => {
        expect(leftReasons(aProgress({ unplacedBy: null }))).toEqual([]);
        expect(leftReasons(aProgress({ unplacedBy: { disagrees: 0 } }))).toEqual([]);
    });
});

describe("what a sync button says while a sync goes on", () => {
    it("names the stage in the middle of a sentence", () => {
        expect(stagePhrase(aProgress({ stage: "render" }))).toBe("visiting the pages");
        expect(said.syncing(stagePhrase(aProgress({ stage: "content" })))).toBe("Syncing · reading the pages");
    });

    it("says the sync is starting before it reached a stage", () => {
        expect(stagePhrase(aProgress({ stage: "", status: "pending" }))).toBe(said.starting);
    });

    it("says the sync is paused when it is", () => {
        expect(stagePhrase(aProgress({ stage: "render", status: "paused" }))).toBe(said.paused);
    });

    it("says the stage is being retried while a failed attempt waits for the next", () => {
        expect(stagePhrase(aProgress({ stage: "render", status: "waiting", attempts: 1 }))).toBe(
            "visiting the pages, retrying",
        );
    });

    it("tells a retry from the pause between two batches", () => {
        expect(syncRetrying(aProgress({ status: "waiting", attempts: 2 }))).toBe(true);
        expect(syncRetrying(aProgress({ status: "waiting", attempts: 0 }))).toBe(false);
        expect(syncRetrying(aProgress({ status: "failed", attempts: 3 }))).toBe(false);
    });
});

describe("the one line a panel shows of a live sync", () => {
    it("names the stage and the pages read so far, and never a total", () => {
        expect(syncLine(aProgress({ stage: "render", pulled: 150 }))).toBe("Visiting the pages · 150 pages read");
        expect(syncLine(aProgress({ stage: "content", pulled: 1 }))).toBe("Reading the pages · 1 page read");
        expect(syncLine(aProgress({ stage: "content", pulled: 1200 }))).toBe("Reading the pages · 1,200 pages read");
    });

    it("counts every page exactly, however many there are", () => {
        expect(syncLine(aProgress({ stage: "content", pulled: 136_100 }))).toBe("Reading the pages · 136,100 pages read");
        expect(pageCount(136_100)).toBe("136,100");
    });
});

describe("the stages of a sync, in order", () => {
    it("lists every stage the walk goes through, in the order it takes them", () => {
        expect(stageSteps(aProgress()).map((step) => step.stage)).toEqual([...syncStages]);
    });

    it("marks the stages behind the current one done and the ones ahead pending", () => {
        const states = stageSteps(aProgress({ stage: "render", status: "running" })).map((step) => step.state);
        expect(states).toEqual(["done", "done", "done", "current", "pending", "pending", "pending", "pending"]);
    });

    it("marks the stage a failed sync stopped in", () => {
        const states = stageSteps(aProgress({ stage: "content", status: "failed" })).map((step) => step.state);
        expect(states).toEqual(["done", "failed", "pending", "pending", "pending", "pending", "pending", "pending"]);
    });

    it("marks the stage a cancelled sync was stopped in", () => {
        const states = stageSteps(aProgress({ stage: "derive", status: "cancelled" })).map((step) => step.state);
        expect(states).toEqual(["done", "done", "done", "done", "stopped", "pending", "pending", "pending"]);
    });

    it("marks every stage done once the sync completed", () => {
        const states = stageSteps(aProgress({ stage: "done", status: "completed" })).map((step) => step.state);
        expect(states).toEqual(Array.from(syncStages, () => "done"));
    });

    it("marks nothing before the walk reached its first stage", () => {
        const states = stageSteps(aProgress({ stage: "", status: "pending" })).map((step) => step.state);
        expect(states).toEqual(Array.from(syncStages, () => "pending"));
    });
});

describe("what a stopped sync is called", () => {
    it("names the stage it stopped in", () => {
        expect(stoppedTitle("render")).toBe("The sync stopped while visiting the pages");
    });

    it("says it stopped before a page was read, or as it finished", () => {
        expect(stoppedTitle("")).toBe(said.stopped.atStart);
        expect(stoppedTitle("done")).toBe(said.stopped.atEnd);
    });
});

describe("a sync refused because another one is under way", () => {
    it("reads the sync under way and the reason from the refusal", () => {
        const message = "a sync of Peptide shop is already under way; it reads every page, so a second one would only repeat it";
        expect(syncConflictOf(refused({ code: "CONFLICT", message, details: { runId: "r-busy" } }))).toEqual({
            message,
            runId: "r-busy",
        });
    });

    it("keeps the reason of a conflict that names no sync", () => {
        expect(syncConflictOf(refused({ code: "CONFLICT", message: "the run moved" }))).toEqual({
            message: "the run moved",
            runId: null,
        });
    });

    it("is no conflict for any other refusal", () => {
        expect(syncConflictOf(refused({ code: "EXTERNAL", message: "WordPress answered 500" }))).toBeNull();
        expect(syncConflictOf(null)).toBeNull();
    });
});

describe("where a sync's run is opened", () => {
    it("is the run's own screen on its site", () => {
        expect(syncRunPath("s1", "r1")).toBe("/s/s1/runs/r1");
    });
});
