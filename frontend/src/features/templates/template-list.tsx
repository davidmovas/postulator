import type { ReactElement } from "react";
import { useMemo } from "react";
import { useNavigate } from "react-router";

import { flatten } from "../../data/call.js";
import { useSite, useUpdateSite } from "../../data/hooks/sites.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { Template } from "../../data/types.js";
import type { TemplateSort } from "../../data/sorts.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { askAgent } from "../agent/dock-state.js";
import {
    Button,
    DenseTable,
    DescriptionIcon,
    EmptyState,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { draftOf } from "./spec.js";

const columns = "minmax(120px,2fr) 96px 80px 96px 64px 96px";
const pageSize = 100;

type TemplateQuery = ReturnType<typeof useTemplates>;

export interface TemplateGroups {
    globals: TemplateQuery;
    locals: TemplateQuery;
    globalRows: readonly Template[];
    localRows: readonly Template[];
    all: readonly Template[];
    loading: boolean;
}

export function useTemplateGroups(siteId: string, sort: TemplateSort | null): TemplateGroups {
    const globals = useTemplates({ scope: "global" }, sort, pageSize);
    const locals = useTemplates({ siteId }, sort, pageSize);
    const globalRows = useMemo(() => flatten(globals.data?.pages), [globals.data]);
    const localRows = useMemo(() => flatten(locals.data?.pages), [locals.data]);
    const all = useMemo(() => [...globalRows, ...localRows], [globalRows, localRows]);
    return { globals, locals, globalRows, localRows, all, loading: globals.isPending || locals.isPending };
}

interface GroupRowProps {
    label: string;
    body: string;
}

function GroupRow({ label, body }: GroupRowProps): ReactElement {
    return (
        <TableRow className="items-start bg-inset py-1" style={{ height: "auto" }}>
            <div role="cell" className="col-span-full flex flex-col gap-0.5">
                <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</span>
                <span className="text-2xs text-ink-faint">{body}</span>
            </div>
        </TableRow>
    );
}

interface TemplateRowProps {
    template: Template;
    isDefault: boolean;
    settingDefault: boolean;
    onOpen: (id: string) => void;
    onMakeDefault: (id: string) => void;
}

function TemplateRow({ template, isDefault, settingDefault, onOpen, onMakeDefault }: TemplateRowProps): ReactElement {
    const draft = useMemo(() => draftOf(template.spec), [template.spec]);
    return (
        <TableRow
            interactive={true}
            tabIndex={0}
            onClick={() => {
                onOpen(template.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter") {
                    event.preventDefault();
                    onOpen(template.id);
                }
            }}
        >
            <TableCell>
                <span className="group flex min-w-0 items-center gap-1.5">
                    <span className="truncate">{template.name}</span>
                    {isDefault ? (
                        <StatusBadge tone="accent" dot={false} className="shrink-0">
                            {copy.templates.siteDefault}
                        </StatusBadge>
                    ) : (
                        <Button
                            size="sm"
                            variant="ghost"
                            busy={settingDefault}
                            className="shrink-0 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                            onClick={(event) => {
                                event.stopPropagation();
                                onMakeDefault(template.id);
                            }}
                        >
                            {copy.templates.editor.makeDefault}
                        </Button>
                    )}
                </span>
            </TableCell>
            <TableCell mono={true} muted={true}>
                {template.pageKind}
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {draft.sections.length}
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {copy.templates.wordRange(draft.lengthMin, draft.lengthMax)}
            </TableCell>
            <TableCell mono={true} align="right" muted={true}>
                {copy.templates.versionLabel(template.version)}
            </TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(template.createdAt)}>
                {relativeTime(template.createdAt)}
            </TableCell>
        </TableRow>
    );
}

interface LoadMoreRowProps {
    busy: boolean;
    onLoad: () => void;
}

function LoadMoreRow({ busy, onLoad }: LoadMoreRowProps): ReactElement {
    return (
        <TableRow className="py-1" style={{ height: "auto" }}>
            <div role="cell" className="col-span-full flex justify-center">
                <Button size="sm" busy={busy} onClick={onLoad}>
                    {copy.app.loadMore}
                </Button>
            </div>
        </TableRow>
    );
}

export interface TemplateListProps {
    siteId: string;
    groups: TemplateGroups;
    sort: TemplateSort | null;
    onSortChange: (sort: TemplateSort | null) => void;
    onCreate: () => void;
}

export function TemplateList({ siteId, groups, sort, onSortChange, onCreate }: TemplateListProps): ReactElement {
    const navigate = useNavigate();
    const site = useSite(siteId);
    const updateSite = useUpdateSite();
    const defaultId = site.data?.site.defaults.templateId ?? null;

    const open = (id: string): void => {
        void navigate(`/s/${siteId}/templates/${id}`);
    };

    const makeDefault = (templateId: string): void => {
        const held = site.data?.site;
        if (held === undefined) {
            return;
        }
        updateSite.mutate({ id: held.id, defaults: { ...held.defaults, templateId } });
    };

    const rowProps = (template: Template) => ({
        template,
        isDefault: template.id === defaultId,
        settingDefault: updateSite.isPending && updateSite.variables?.defaults?.templateId === template.id,
        onOpen: open,
        onMakeDefault: makeDefault,
    });

    const toggle = (field: TemplateSort["field"]): void => {
        if (sort === null || sort.field !== field) {
            onSortChange({ field, desc: false });
            return;
        }
        onSortChange(sort.desc ? null : { field, desc: true });
    };

    if (groups.loading) {
        return (
            <div className="p-3">
                <SkeletonRows rows={10} label={copy.templates.loading} />
            </div>
        );
    }

    if (groups.all.length === 0) {
        return (
            <div className="flex items-start justify-center p-6">
                <EmptyState
                    icon={DescriptionIcon}
                    title={copy.templates.nothingToCopy}
                    body={copy.empty.templates}
                    actions={
                        <>
                            <Button variant="primary" onClick={onCreate}>
                                {copy.templates.newTemplate}
                            </Button>
                            <Button
                                onClick={() => {
                                    askAgent(copy.agent.ask.newTemplate);
                                }}
                            >
                                {copy.templates.askAgent}
                            </Button>
                        </>
                    }
                />
            </div>
        );
    }

    return (
        <div>
            <DenseTable columns={columns} label={copy.templates.title}>
                <TableHead>
                    <SortableHeader
                        active={sort?.field === "name"}
                        direction={sort?.desc === true ? "desc" : "asc"}
                        onToggle={() => {
                            toggle("name");
                        }}
                    >
                        {copy.templates.columns.name}
                    </SortableHeader>
                    <div>{copy.templates.columns.pageKind}</div>
                    <div className="text-right">{copy.templates.columns.sections}</div>
                    <div className="text-right">{copy.templates.columns.words}</div>
                    <div className="text-right">{copy.templates.columns.version}</div>
                    <SortableHeader
                        active={sort?.field === "createdAt"}
                        direction={sort?.desc === true ? "desc" : "asc"}
                        onToggle={() => {
                            toggle("createdAt");
                        }}
                    >
                        {copy.templates.columns.created}
                    </SortableHeader>
                </TableHead>
                {groups.globalRows.length === 0 ? null : (
                    <>
                        <GroupRow label={copy.templates.groups.global} body={copy.templates.groups.globalBody} />
                        {groups.globalRows.map((template) => (
                            <TemplateRow key={template.id} {...rowProps(template)} />
                        ))}
                        {groups.globals.hasNextPage ? (
                            <LoadMoreRow
                                busy={groups.globals.isFetchingNextPage}
                                onLoad={() => {
                                    void groups.globals.fetchNextPage();
                                }}
                            />
                        ) : null}
                    </>
                )}
                {groups.localRows.length === 0 ? null : (
                    <>
                        <GroupRow label={copy.templates.groups.site} body={copy.templates.groups.siteBody} />
                        {groups.localRows.map((template) => (
                            <TemplateRow key={template.id} {...rowProps(template)} />
                        ))}
                        {groups.locals.hasNextPage ? (
                            <LoadMoreRow
                                busy={groups.locals.isFetchingNextPage}
                                onLoad={() => {
                                    void groups.locals.fetchNextPage();
                                }}
                            />
                        ) : null}
                    </>
                )}
            </DenseTable>
        </div>
    );
}
