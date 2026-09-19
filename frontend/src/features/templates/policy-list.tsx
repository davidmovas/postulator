import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { flatten } from "../../data/call.js";
import { failure } from "../../data/errors.js";
import { useSite } from "../../data/hooks/sites.js";
import { useDeletePolicy, useEffectivePolicy, usePolicies } from "../../data/hooks/templates.js";
import type { LinkPolicy } from "../../data/types.js";
import type { PolicySort } from "../../data/sorts.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    AddIcon,
    Banner,
    Button,
    DeleteIcon,
    DenseTable,
    Dialog,
    EditNoteIcon,
    EmptyState,
    IconButton,
    LinkIcon,
    Panel,
    PanelHeader,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
} from "../../ui/index.js";
import { anchorLabel, forbidsLabel, scopeLabel, scopeTone } from "./labels.js";
import { PolicyDrawer } from "./policy-form.js";

const columns = "minmax(120px,2fr) 96px 120px minmax(96px,1.4fr) 96px 64px";
const pageSize = 100;

interface EffectivePanelProps {
    siteId: string;
}

function EffectivePanel({ siteId }: EffectivePanelProps): ReactElement {
    const site = useSite(siteId);
    const effective = useEffectivePolicy(siteId);
    const policy = effective.data?.policy;
    const chosen = site.data?.site.defaults.linkPolicyId ?? null;

    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.policies.effective} />
            <div className="flex flex-col gap-3 p-3">
                <Banner tone="info" icon={LinkIcon} title={copy.policies.split} body={copy.policies.splitBody} />
                {effective.isPending ? (
                    <SkeletonRows rows={3} label={copy.policies.loading} />
                ) : policy === undefined ? (
                    <p className="text-xs text-warn">
                        {failure(effective.error).code === "NOT_FOUND"
                            ? copy.policies.effectiveMissing
                            : failure(effective.error).message}
                    </p>
                ) : (
                    <div className="flex flex-col gap-2">
                        <div className="flex flex-wrap items-center gap-2">
                            <span className="text-sm font-semibold text-ink">{policy.name}</span>
                            <StatusBadge tone={scopeTone(policy.scope)} dot={false}>
                                {scopeLabel(policy.scope)}
                            </StatusBadge>
                        </div>
                        <p className="text-xs text-ink-dim">
                            {chosen === policy.id ? copy.policies.effectiveFromSite : copy.policies.effectiveFallback}
                        </p>
                        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs">
                            <dt className="text-ink-dim">{copy.policies.forbidExternal}</dt>
                            <dd className="font-mono text-ink">{policy.forbidExternal ? "yes" : "no"}</dd>
                            <dt className="text-ink-dim">{copy.policies.forbidSelf}</dt>
                            <dd className="font-mono text-ink">{policy.forbidSelf ? "yes" : "no"}</dd>
                            <dt className="text-ink-dim">{copy.policies.anchorStrategy}</dt>
                            <dd className="text-ink">{anchorLabel(policy.anchorStrategy)}</dd>
                        </dl>
                    </div>
                )}
            </div>
        </Panel>
    );
}

interface GroupRowProps {
    label: string;
}

function GroupRow({ label }: GroupRowProps): ReactElement {
    return (
        <TableRow className="bg-inset" style={{ height: "auto" }}>
            <div role="cell" className="col-span-full py-1 text-2xs font-semibold tracking-label text-ink-faint uppercase">
                {label}
            </div>
        </TableRow>
    );
}

interface PolicyRowProps {
    policy: LinkPolicy;
    inForce: boolean;
    onEdit: (policy: LinkPolicy) => void;
    onDelete: (policy: LinkPolicy) => void;
}

function PolicyRow({ policy, inForce, onEdit, onDelete }: PolicyRowProps): ReactElement {
    return (
        <TableRow>
            <TableCell>
                <span className="flex min-w-0 items-center gap-1.5">
                    <span className="truncate">{policy.name}</span>
                    {inForce ? (
                        <StatusBadge tone="ok" dot={false} className="shrink-0">
                            {copy.policies.effective}
                        </StatusBadge>
                    ) : null}
                </span>
            </TableCell>
            <TableCell muted={true}>{scopeLabel(policy.scope)}</TableCell>
            <TableCell mono={true} muted={true}>
                {forbidsLabel(policy.forbidExternal, policy.forbidSelf)}
            </TableCell>
            <TableCell muted={true}>{anchorLabel(policy.anchorStrategy)}</TableCell>
            <TableCell mono={true} muted={true} title={absoluteTime(policy.createdAt)}>
                {relativeTime(policy.createdAt)}
            </TableCell>
            <TableCell className="flex justify-end gap-1">
                <IconButton
                    icon={EditNoteIcon}
                    label={copy.policies.edit}
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                        onEdit(policy);
                    }}
                />
                <IconButton
                    icon={DeleteIcon}
                    label={copy.policies.delete}
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                        onDelete(policy);
                    }}
                />
            </TableCell>
        </TableRow>
    );
}

export interface PolicyListProps {
    siteId: string;
    sort: PolicySort | null;
    onSortChange: (sort: PolicySort | null) => void;
}

export function PolicyList({ siteId, sort, onSortChange }: PolicyListProps): ReactElement {
    const globals = usePolicies({ scope: "global" }, sort, pageSize);
    const locals = usePolicies({ siteId }, sort, pageSize);
    const effective = useEffectivePolicy(siteId);
    const remove = useDeletePolicy();
    const [editing, setEditing] = useState<LinkPolicy | null>(null);
    const [creating, setCreating] = useState(false);
    const [doomed, setDoomed] = useState<LinkPolicy | null>(null);

    const globalRows = useMemo(() => flatten(globals.data?.pages), [globals.data]);
    const localRows = useMemo(() => flatten(locals.data?.pages), [locals.data]);
    const inForceId = effective.data?.policy.id ?? null;
    const seed = effective.data?.policy ?? null;

    const toggle = (field: PolicySort["field"]): void => {
        if (sort === null || sort.field !== field) {
            onSortChange({ field, desc: false });
            return;
        }
        onSortChange(sort.desc ? null : { field, desc: true });
    };

    const rows = (list: readonly LinkPolicy[]): ReactElement[] =>
        list.map((policy) => (
            <PolicyRow
                key={policy.id}
                policy={policy}
                inForce={policy.id === inForceId}
                onEdit={setEditing}
                onDelete={setDoomed}
            />
        ));

    return (
        <div className="flex flex-col gap-3 p-3">
            <EffectivePanel siteId={siteId} />
            <Panel className="min-w-0">
                <PanelHeader title={copy.policies.title}>
                    <Button
                        size="sm"
                        icon={AddIcon}
                        disabled={seed === null}
                        onClick={() => {
                            setCreating(true);
                        }}
                    >
                        {copy.policies.newPolicy}
                    </Button>
                </PanelHeader>
                {globals.isPending || locals.isPending ? (
                    <div className="p-3">
                        <SkeletonRows rows={6} label={copy.policies.loading} />
                    </div>
                ) : globalRows.length === 0 && localRows.length === 0 ? (
                    <div className="p-4">
                        <EmptyState
                            icon={LinkIcon}
                            title={copy.policies.title}
                            body={copy.empty.policies}
                            actions={
                                <Button
                                    variant="primary"
                                    icon={AddIcon}
                                    disabled={seed === null}
                                    onClick={() => {
                                        setCreating(true);
                                    }}
                                >
                                    {copy.policies.newPolicy}
                                </Button>
                            }
                        />
                    </div>
                ) : (
                    <DenseTable columns={columns} label={copy.policies.title}>
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
                            <div>{copy.templates.columns.forbids}</div>
                            <div>{copy.templates.columns.anchors}</div>
                            <SortableHeader
                                active={sort?.field === "createdAt"}
                                direction={sort?.desc === true ? "desc" : "asc"}
                                onToggle={() => {
                                    toggle("createdAt");
                                }}
                            >
                                {copy.templates.columns.created}
                            </SortableHeader>
                            <div />
                        </TableHead>
                        {globalRows.length === 0 ? null : (
                            <>
                                <GroupRow label={copy.templates.groups.global} />
                                {rows(globalRows)}
                            </>
                        )}
                        {localRows.length === 0 ? null : (
                            <>
                                <GroupRow label={copy.templates.groups.site} />
                                {rows(localRows)}
                            </>
                        )}
                    </DenseTable>
                )}
            </Panel>
            {creating && seed !== null ? (
                <PolicyDrawer
                    siteId={siteId}
                    policy={null}
                    seed={seed}
                    onClose={() => {
                        setCreating(false);
                    }}
                />
            ) : null}
            {editing !== null && seed !== null ? (
                <PolicyDrawer
                    key={editing.id}
                    siteId={siteId}
                    policy={editing}
                    seed={seed}
                    onClose={() => {
                        setEditing(null);
                    }}
                />
            ) : null}
            <Dialog
                open={doomed !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setDoomed(null);
                    }
                }}
                title={copy.policies.deleteTitle}
                description={copy.policies.deleteBody}
                confirmLabel={copy.policies.deleteConfirm}
                cancelLabel={copy.policies.create.cancel}
                destructive={true}
                icon={DeleteIcon}
                busy={remove.isPending}
                onConfirm={() => {
                    if (doomed === null) {
                        return;
                    }
                    remove.mutate(
                        { id: doomed.id },
                        {
                            onSuccess: () => {
                                setDoomed(null);
                            },
                        },
                    );
                }}
            />
        </div>
    );
}
