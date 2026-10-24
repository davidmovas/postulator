export const findings = {
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
    },
};
