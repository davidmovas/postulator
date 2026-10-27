function numbered(numbers: readonly number[]): string {
    const named = numbers.map(String);
    if (named.length <= 1) {
        return named.join("");
    }
    return `${named.slice(0, -1).join(", ")} and ${named[named.length - 1] ?? ""}`;
}

export const plans = {
    kicker: "Plan",
    group: (title: string) => `The plan ${title}`,
    steps: (count: number) => (count === 1 ? "1 step" : `${count} steps`),
    statuses: {
        proposed: "Waiting for your approval",
        running: "Running",
        done: "Finished",
        declined: "Declined",
    },
    stepStatuses: {
        proposed: "Proposed",
        declined: "Not approved",
        pending: "Queued",
        running: "Running",
        waiting: "Waiting on its run",
        done: "Done",
        failed: "Failed",
        skipped: "Skipped",
    },
    after: (numbers: readonly number[]) =>
        numbers.length === 1 ? `After step ${numbered(numbers)}` : `After steps ${numbered(numbers)}`,
    tick: (number: number) => `Approve step ${number}`,
    tickDependents: (count: number) =>
        count === 1
            ? "Unticking it unticks the 1 step that waits on it."
            : `Unticking it unticks the ${count} steps that wait on it.`,
    estimate: (usd: string, tokens: string) => `about ${usd}, ${tokens} tokens`,
    total: (chosen: number, all: number, usd: string, priced: number) => {
        const picked = chosen === all ? `All ${all} steps chosen` : `${chosen} of ${all} steps chosen`;
        return priced === 0 ? picked : `${picked}, about ${usd} in model calls`;
    },
    nothingChosen: "No step is ticked, so approving would run nothing.",
    approveAll: "Approve all",
    approveChosen: (count: number) => (count === 1 ? "Approve 1 chosen" : `Approve ${count} chosen`),
    approveChosenHint: "Untick the steps you do not want, then approve the rest.",
    decline: "Decline",
    approving: "Approving",
    declining: "Declining",
    editsTitle: "Change before approving",
    editsHint: "Only what you change is sent; the step is checked again before it runs.",
    editNames: {
        anchor: "Anchor",
        value: "Value",
        goal: "Goal",
        focus: "Focus",
        tone: "Tone",
        avoid: "Avoid",
        notes: "Notes for the page",
        reference: "Reference text",
        styleSample: "Style sample",
        title: "Title",
        name: "Name",
        type: "Page type",
        pageTypeId: "Page type",
        templateId: "Template",
    } as Readonly<Record<string, string>>,
    openRun: "Open its run",
    reasonLabel: "Why",
    settled: "This plan was settled elsewhere, so it is read again.",
    loading: "Reading the plan",
    declinedBody: "None of its steps ran.",
    finishedBody: (done: number, all: number) =>
        done === all ? "Every approved step landed." : `${done} of ${all} approved steps landed.`,
};
