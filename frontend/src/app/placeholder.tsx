import type { ReactElement } from "react";

import { EmptyState, Screen } from "../ui/index.js";
import type { IconComponent } from "../ui/index.js";

export interface PlaceholderProps {
    title: string;
    icon: IconComponent;
}

export function Placeholder({ title, icon }: PlaceholderProps): ReactElement {
    return (
        <Screen title={title}>
            <div className="flex h-full items-center justify-center">
                <EmptyState icon={icon} title={title} className="w-80" />
            </div>
        </Screen>
    );
}
