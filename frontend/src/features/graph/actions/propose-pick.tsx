import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import type { SegmentedOption, SelectOption } from "../../../ui/index.js";
import { Field, Segmented, Select, Textarea } from "../../../ui/index.js";
import { everyNode } from "../../pages/pick/model.js";
import type { TreeView } from "../../pages/pick/tree.js";
import { PageTree, blankTreeView } from "../../pages/pick/tree.js";
import type { GraphIndex } from "../model/index.js";
import { keywordLines, unmapped } from "../model/proposals.js";

const rootPath = "/";

export const noParent = "";

export type ProposalInput = "pages" | "keywords";

type PickScope = "unmapped" | "all";

const scopeOptions: readonly SegmentedOption<PickScope>[] = [
    { value: "unmapped", label: copy.graph.ai.pickScopes.unmapped },
    { value: "all", label: copy.graph.ai.pickScopes.all },
];

const inputOptions: readonly SegmentedOption<ProposalInput>[] = [
    { value: "pages", label: copy.graph.ai.sourcePages },
    { value: "keywords", label: copy.graph.ai.sourceKeywords },
];

export interface Pick {
    input: ProposalInput;
    selected: ReadonlySet<string>;
    scope: PickScope;
    keywords: string;
    parentId: string;
    tree: TreeView;
}

export function freshPick(input: ProposalInput): Pick {
    return { input, selected: new Set(), scope: "unmapped", keywords: "", parentId: noParent, tree: blankTreeView };
}

export interface ProposalPickProps {
    siteId: string;
    index: GraphIndex;
    pick: Pick;
    onPick: (next: Pick) => void;
}

export function ProposalPick({ siteId, index, pick, onPick }: ProposalPickProps): ReactElement {
    const keep = useMemo(() => (pick.scope === "unmapped" ? unmapped : everyNode), [pick.scope]);
    const lines = useMemo(() => keywordLines(pick.keywords), [pick.keywords]);
    const parents: SelectOption<string>[] = useMemo(
        () => [
            { value: noParent, label: copy.graph.ai.parentNone },
            ...[...index.byId.values()].map((entity) => ({ value: entity.id, label: entity.name })),
        ],
        [index],
    );

    return (
        <>
            <Segmented
                label={copy.graph.ai.entitiesTitle}
                value={pick.input}
                options={inputOptions}
                onValueChange={(input) => {
                    onPick({ ...pick, input });
                }}
            />
            {pick.input === "pages" ? (
                <PageTree
                    siteId={siteId}
                    selected={pick.selected}
                    onChange={(selected) => {
                        onPick({ ...pick, selected });
                    }}
                    pickable={unmapped}
                    keep={keep}
                    fill={true}
                    view={pick.tree}
                    onView={(tree) => {
                        onPick({ ...pick, tree });
                    }}
                    label={copy.graph.ai.pickPages}
                    picked={copy.graph.ai.pickedPages(pick.selected.size)}
                    refusalOf={(node) => (node.page.path === rootPath ? copy.graph.ai.root : unmapped(node) ? null : copy.graph.ai.mapped)}
                    filter={
                        <Segmented
                            label={copy.graph.ai.pickScope}
                            size="sm"
                            value={pick.scope}
                            options={scopeOptions}
                            onValueChange={(scope) => {
                                onPick({ ...pick, scope });
                            }}
                        />
                    }
                />
            ) : (
                <>
                    <Field label={copy.graph.ai.keywords} hint={copy.graph.ai.keywordsHint}>
                        {(control) => (
                            <Textarea
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                rows={16}
                                value={pick.keywords}
                                onChange={(event) => {
                                    onPick({ ...pick, keywords: event.target.value });
                                }}
                            />
                        )}
                    </Field>
                    <div className="flex items-center justify-between gap-2">
                        <span className="font-mono text-2xs text-ink-faint">{copy.graph.ai.keywordsCount(lines.length)}</span>
                        <div className="w-64">
                            <Field label={copy.graph.ai.parentLabel}>
                                {(control) => (
                                    <Select
                                        id={control.id}
                                        aria-describedby={control["aria-describedby"]}
                                        value={pick.parentId}
                                        options={parents}
                                        onValueChange={(parentId) => {
                                            onPick({ ...pick, parentId });
                                        }}
                                    />
                                )}
                            </Field>
                        </div>
                    </div>
                </>
            )}
        </>
    );
}
