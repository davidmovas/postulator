const quoted = (text: string): string => `“${text}”`;

export const findings = {
    title: "Findings",
    loading: "Reading the findings",
    codes: "Codes",
    all: "All",
    status: "Status",
    statuses: {
        open: "Open",
        dismissed: "Dismissed",
        resolved: "Resolved",
    },
    type: "Page type",
    everyType: "Every type",
    under: "Under a page",
    underTooltip: "Show only the findings of one page and every page below it.",
    underRemove: (path: string) => `Show the findings of every page, not only under ${path}`,
    site: "The whole site",
    openPage: "Open the page in Pages",
    more: "Show more",
    pick: {
        title: "Findings under a page",
        body: "Pick a page: its findings and those of every page below it stay listed.",
        confirm: "Show its findings",
        cancel: "Cancel",
    },
    search: {
        label: "Find a page",
        hint: "Title or path",
        none: "No page matches.",
        typeMore: "Type a title or a path.",
    },
    settled: (by: string, when: string, how: string) => `Settled by ${by} ${when}: ${how}`,
    resolvedBy: {
        person: "a person",
        agent: "the agent",
        observation: "the observation",
        import: "an import",
        model: "the model",
    } as Readonly<Record<string, string>>,
    someone: "someone",
    resolutions: {
        ignore: "ignored",
        accept: "the card was written",
        accept_site: "the site's value was taken",
        accept_workbook: "the workbook's parent was taken",
        map_to_page: "the row was mapped to a page",
        set_type: "the page was given a type",
        rewrite_field: "the field was written again",
        relink_page: "the page was relinked",
        rescan: "the page was fetched again",
        propose_entity: "an entity was proposed for the page",
        page_mapped: "an entity now carries the page",
        observed_clear: "the site no longer shows it",
        row_matched: "the row found its page",
        row_left_sheet: "the row left the workbook",
    } as Readonly<Record<string, string>>,
    unknownResolution: "settled",
    resolve: {
        body: "Confirming settles this finding as yours.",
        confirm: "Confirm",
        cancel: "Cancel",
        reading: "Reading what changes",
        gone: "This finding is no longer open, so the list is read again.",
        typeLabel: "New type",
        typeFirst: "Pick a type first.",
        pageLabel: "The row's page",
        candidates: "Pages the row matches",
        pageFirst: "Pick a page first.",
    },
    bulk: {
        all: (label: string, count: number) => `${label} all ${count}`,
        tooltip: "Settles every finding of this code the filters show, in one confirmed action.",
        title: (label: string) => `${label}: every finding of this code`,
        body: (count: number) =>
            count === 1 ? "Confirming settles 1 finding as yours." : `Confirming settles ${count} findings as yours.`,
        collecting: "Gathering every finding of this code",
        confirm: (count: number) => (count === 1 ? "Settle 1 finding" : `Settle ${count} findings`),
        partial: (settled: number, total: number, reason: string) =>
            `${settled} of ${total} findings were settled; the other ${total - settled} were not: ${reason}${/[.!?]$/.test(reason) ? "" : "."} Confirm again to settle the ones still open.`,
        propose: (count: number) => (count === 1 ? "Propose an entity for 1 page" : `Propose entities for all ${count}`),
        proposeTooltip:
            "Opens entity proposals named from the site for every page of this code the filters show; nothing is settled.",
    },
    cards: "Accept it on the Graph screen, where the proposed card is shown whole.",
    empty: {
        openTitle: "Nothing to settle",
        openBody: "The observation and the imports found nothing here that disagrees with the model.",
        settledTitle: "Nothing settled yet",
        settledBody: "A finding settled by a person, the agent, the observation or an import is listed here.",
        filtered: "Show every finding",
    },
    options: {
        ignore: "Ignore",
        accept: "Accept the card",
        accept_site: "Take the site's",
        accept_workbook: "Take the workbook's",
        map_to_page: "Map to a page",
        set_type: "Set the type",
        rewrite_field: "Rewrite it",
        relink_page: "Relink the page",
        rescan: "Fetch again",
        propose_entity: "Propose an entity",
    },
    optionDoes: {
        ignore: "Dismisses the finding, so the model keeps what it holds; nothing is written to the site.",
        accept: "Writes the proposed card to its entity in the model; nothing is written to the site.",
        accept_site: "The model takes what the site shows now as its own; nothing is written to the site.",
        accept_workbook: "The page takes the workbook's parent in the model; nothing is written to the site.",
        map_to_page: "Ties the workbook row to the page you pick in the model; nothing is written to the site.",
        set_type: "Gives the page the type you pick in the model; nothing is written to the site.",
        rewrite_field: "Opens a run start that writes the field again over the client's value.",
        relink_page: "Opens a run start that places the page's owed links again.",
        rescan: "Opens a sync that walks every page and fetches each render again.",
        propose_entity:
            "Opens entity proposals named from the site, which the model takes only once you accept them; nothing is written to the site.",
    },
    subjects: {
        unresolved: (name: string, path: string) => {
            if (name === "") {
                return "A crumb of the breadcrumb leads to no page in the model.";
            }
            return path === ""
                ? `The breadcrumb names ${quoted(name)}, which leads to no page in the model.`
                : `The breadcrumb names ${quoted(name)} at ${path}, which is no page in the model.`;
        },
        ambiguous: (name: string, candidates: readonly string[]) => {
            if (name === "" || candidates.length === 0) {
                return "A crumb of the breadcrumb matches several pages.";
            }
            return `The breadcrumb names ${quoted(name)}, which ${candidates.length} pages carry: ${candidates.join(", ")}.`;
        },
        twin: (name: string, page: string, terms: readonly string[]) => {
            if (name === "" || page === "") {
                return "A crumb's name is shared by a page and a store category, so the page was taken.";
            }
            const over = terms.length === 0 ? "a store category of the same name" : `the store category ${terms.join(", ")}`;
            return `The breadcrumb names ${quoted(name)}: the page ${page} was taken over ${over}.`;
        },
        cycle: (paths: readonly string[]) =>
            paths.length === 0
                ? "The parents loop, so the loop is broken here."
                : `The parents loop through ${paths.join(" › ")}, so the loop is broken here.`,
        differs: (sheet: readonly string[], site: readonly string[]) => {
            if (sheet.length === 0) {
                return "The workbook places this page under other parents than the site shows.";
            }
            const shown = site.length === 0 ? "at the top" : `under ${site.join(" › ")}`;
            return `The workbook puts this page under ${sheet.join(" › ")}; the site shows it ${shown}.`;
        },
        rowAmbiguous: (sheet: string, candidates: readonly string[]) => {
            const from = sheet === "" ? "the workbook" : `the sheet ${sheet}`;
            return candidates.length === 0
                ? `A row of ${from} matches several pages.`
                : `A row of ${from} matches ${candidates.length} pages: ${candidates.join(", ")}.`;
        },
        rowWithoutPage: (sheet: string, planned: string, product: boolean) => {
            const from = sheet === "" ? "the workbook" : `the sheet ${sheet}`;
            if (product) {
                return `A product row of ${from} waits for its product in the store.`;
            }
            return planned === ""
                ? `A row of ${from} has no page on the site yet.`
                : `A row of ${from} has no page on the site yet; it is planned at ${planned}.`;
        },
        outdated: (version: string, required: string) =>
            required === ""
                ? "The companion plugin is out of date."
                : `The companion plugin is version ${version === "" ? "unknown" : version} and ${required} is needed.`,
        field: (field: string) => `The ${field} was edited on the site since it was last read or written.`,
        body: "The body was edited on the site since it was last read or written.",
        unreachable: (status: number) =>
            status > 0
                ? `The site answered ${status} when the page was fetched.`
                : "The page could not be fetched from the site.",
        redirected: (target: string) =>
            target === "" ? "Visiting the page landed on another address." : `Visiting the page landed on ${target}.`,
        builder: (name: string) => `Made with ${name}, so its body is read only.`,
        technical: (reason: string) => `Read as a technical page: ${reason}.`,
        titleDiffers: (planned: string, site: string) => `The plan says ${quoted(planned)}, the site says ${quoted(site)}.`,
        urlForms: (absolute: number, relative: number) =>
            `The site's pages mix link forms: ${absolute} absolute, ${relative} relative.`,
        cardProposed: (name: string) =>
            name === ""
                ? "The agent proposed a card for an entity; accept or dismiss it on the Graph screen."
                : `The agent proposed a card for ${name}; accept or dismiss it on the Graph screen.`,
        withoutEntity: (path: string) =>
            path === ""
                ? "No entity carries this page, so no run can write it."
                : `No entity carries ${path}, so no run can write it.`,
        generic: "The observation flagged something here.",
    },
};
