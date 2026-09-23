import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { AwaitedParent } from "../../data/types.js";
import { AccountTreeIcon, Banner, Button, OpenInNewIcon, RefreshIcon } from "../../ui/index.js";
import { holdBody, parentRegenerable } from "./hold.js";

export interface ParentHoldProps {
    parent: AwaitedParent;
    busy: boolean;
    onOpenItem: (itemId: string) => void;
    onRegenerate: (itemId: string) => void;
}

export function ParentHold({ parent, busy, onOpenItem, onRegenerate }: ParentHoldProps): ReactElement {
    const inRun = parent.itemId !== "";
    return (
        <Banner
            tone="info"
            icon={AccountTreeIcon}
            title={copy.runs.hold.title(parent.path)}
            body={holdBody(parent)}
            actions={
                inRun ? (
                    <>
                        <Button
                            size="sm"
                            icon={OpenInNewIcon}
                            onClick={() => {
                                onOpenItem(parent.itemId);
                            }}
                        >
                            {copy.runs.hold.openParent(parent.path)}
                        </Button>
                        {parentRegenerable(parent) ? (
                            <Button
                                size="sm"
                                variant="primary"
                                icon={RefreshIcon}
                                busy={busy}
                                onClick={() => {
                                    onRegenerate(parent.itemId);
                                }}
                            >
                                {copy.runs.hold.regenerateParent(parent.path)}
                            </Button>
                        ) : null}
                    </>
                ) : undefined
            }
        />
    );
}
