import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useDeleteSite } from "../../data/hooks/sites.js";
import type { Site } from "../../data/types.js";
import { Banner, DeleteForeverIcon, Dialog } from "../../ui/index.js";
import { BusyBanner, isSiteBusy, quietProblem } from "./clear-dialog.js";

export interface SiteDeleteDialogProps {
    site: Site;
    onClose: () => void;
    onDeleted: (id: string) => void;
}

export function SiteDeleteDialog({ site, onClose, onDeleted }: SiteDeleteDialogProps): ReactElement {
    const remove = useDeleteSite();
    const busy = isSiteBusy(remove.error);
    const problem = quietProblem(remove.error, ["CONFLICT"]);

    return (
        <Dialog
            open={true}
            onOpenChange={(next) => {
                if (!next) {
                    onClose();
                }
            }}
            title={copy.sites.deleteTitle}
            description={copy.sites.deleteBody(site.name)}
            icon={DeleteForeverIcon}
            destructive={true}
            confirmLabel={copy.sites.deleteConfirm}
            cancelLabel={copy.app.cancel}
            busy={remove.isPending}
            onConfirm={() => {
                remove.mutate(
                    { id: site.id },
                    {
                        onSuccess: () => {
                            onDeleted(site.id);
                            onClose();
                        },
                    },
                );
            }}
        >
            {busy ? <BusyBanner siteId={site.id} /> : null}
            {problem === null ? null : <Banner tone="danger" title={problem} />}
        </Dialog>
    );
}
