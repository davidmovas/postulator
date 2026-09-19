import { copy } from "../copy/index.js";
import { EmptyState, PendingActionsIcon } from "../ui/index.js";

export interface NotBuiltProps {
    screen: string;
    wave: string;
}

export function NotBuilt({ screen, wave }: NotBuiltProps) {
    return (
        <div className="p-6">
            <EmptyState
                icon={PendingActionsIcon}
                title={`${screen} — ${copy.notBuilt.title}`}
                body={`${copy.notBuilt.body} ${copy.notBuilt.wave(wave)}`}
                className="max-w-lg"
            />
        </div>
    );
}
