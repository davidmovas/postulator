import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { PageModel } from "../../../data/types.js";
import { Panel, PanelHeader } from "../../../ui/index.js";
import { Fact, Facts } from "./facts.js";
import { readOnlyReasonLabel, technicalReasonLabel } from "./labels.js";

const said = copy.pages.model.reasons;

export interface ReasonsPaneProps {
    model: PageModel;
}

export function ReasonsPane({ model }: ReasonsPaneProps): ReactElement | null {
    if (model.technicalReason === "" && model.readOnlyReason === "") {
        return null;
    }
    return (
        <Panel>
            <PanelHeader title={said.title} />
            <Facts>
                {model.technicalReason === "" ? null : (
                    <Fact label={said.technical}>{technicalReasonLabel(model.technicalReason)}</Fact>
                )}
                {model.readOnlyReason === "" ? null : (
                    <Fact label={said.readOnly}>{readOnlyReasonLabel(model.readOnlyReason)}</Fact>
                )}
            </Facts>
        </Panel>
    );
}
