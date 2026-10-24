import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { Button, EmptyState, HubIcon, UploadFileIcon } from "../../ui/index.js";

export interface GraphEmptyProps {
    onPropose: () => void;
    onImport: () => void;
}

export function GraphEmpty({ onPropose, onImport }: GraphEmptyProps): ReactElement {
    return (
        <div className="flex h-full items-center justify-center">
            <EmptyState
                icon={HubIcon}
                title={copy.graph.empty.title}
                body={copy.graph.empty.body}
                actions={
                    <>
                        <Button variant="primary" onClick={onPropose}>
                            {copy.graph.empty.propose}
                        </Button>
                        <Button variant="secondary" icon={UploadFileIcon} onClick={onImport}>
                            {copy.graph.empty.import}
                        </Button>
                    </>
                }
            />
        </div>
    );
}
