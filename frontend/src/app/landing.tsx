import type { ReactElement } from "react";
import { Navigate } from "react-router";

import { copy } from "../copy/index.js";
import { flatten } from "../data/call.js";
import { useSites } from "../data/hooks/sites.js";
import { Spinner } from "../ui/index.js";
import { landing, readLastSite } from "./site-memory.js";

export function Landing(): ReactElement {
    const sites = useSites();

    if (sites.isPending) {
        return (
            <div className="flex h-full items-center justify-center gap-2 text-xs text-ink-dim">
                <Spinner size={12} />
                {copy.app.loading}
            </div>
        );
    }

    return <Navigate to={landing(readLastSite(), flatten(sites.data?.pages).map((site) => site.id))} replace={true} />;
}
