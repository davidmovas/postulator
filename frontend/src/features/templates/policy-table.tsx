import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { LinkPolicy } from "../../data/types.js";
import {
    Button,
    DenseTable,
    EmptyState,
    LinkIcon,
    SkeletonRows,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    VerifiedIcon,
} from "../../ui/index.js";
import { anchorLabel, forbidsLabel, scopeLabel, scopeTone } from "./labels.js";

const columns = "minmax(140px,1fr) 96px 96px 112px minmax(96px,1.2fr) 80px";

interface RowProps {
    policy: LinkPolicy;
    selected: boolean;
    inForce: boolean;
    onSelect: (id: string) => void;
}

function Row({ policy, selected, inForce, onSelect }: RowProps): ReactElement {
    return (
        <TableRow
            interactive={true}
            selected={selected}
            tabIndex={0}
            data-policy-row={true}
            data-policy-id={policy.id}
            onClick={() => {
                onSelect(policy.id);
            }}
            onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onSelect(policy.id);
                }
            }}
        >
            <TableCell>
                <span className="truncate font-medium text-ink">{policy.name}</span>
            </TableCell>
            <TableCell>
                <StatusBadge tone={scopeTone(policy.scope)} dot={false}>
                    {scopeLabel(policy.scope)}
                </StatusBadge>
            </TableCell>
            <TableCell>
                {inForce ? (
                    <StatusBadge tone="accent" icon={VerifiedIcon}>
                        {copy.policies.inForce}
                    </StatusBadge>
                ) : null}
            </TableCell>
            <TableCell muted={true}>{forbidsLabel(policy.forbidExternal, policy.forbidSelf)}</TableCell>
            <TableCell muted={true}>{anchorLabel(policy.anchorStrategy)}</TableCell>
            <TableCell mono={true} muted={true} align="right">
                {policy.rules.maxLinks}
            </TableCell>
        </TableRow>
    );
}

export interface PolicyTableProps {
    rows: readonly LinkPolicy[];
    pending: boolean;
    selectedId: string | null;
    inForceId: string | null;
    hasMore: boolean;
    loadingMore: boolean;
    onSelect: (id: string) => void;
    onLoadMore: () => void;
    onCreate: () => void;
}

export function PolicyTable({
    rows,
    pending,
    selectedId,
    inForceId,
    hasMore,
    loadingMore,
    onSelect,
    onLoadMore,
    onCreate,
}: PolicyTableProps): ReactElement {
    if (pending) {
        return (
            <div className="p-4">
                <SkeletonRows rows={6} label={copy.policies.loading} />
            </div>
        );
    }

    if (rows.length === 0) {
        return (
            <div className="flex items-start justify-center p-6">
                <EmptyState
                    icon={LinkIcon}
                    title={copy.policies.empty}
                    body={copy.policies.emptyBody}
                    actions={
                        <Button variant="primary" onClick={onCreate}>
                            {copy.policies.newPolicy}
                        </Button>
                    }
                />
            </div>
        );
    }

    return (
        <div className="min-h-0 flex-1 overflow-auto">
            <DenseTable columns={columns} label={copy.policies.title}>
                <TableHead>
                    <div>{copy.policies.columns.name}</div>
                    <div>{copy.policies.columns.scope}</div>
                    <div>{copy.policies.columns.inForce}</div>
                    <div>{copy.policies.columns.forbids}</div>
                    <div>{copy.policies.columns.anchor}</div>
                    <div className="text-right">{copy.policies.columns.maxLinks}</div>
                </TableHead>
                {rows.map((policy) => (
                    <Row
                        key={policy.id}
                        policy={policy}
                        selected={policy.id === selectedId}
                        inForce={policy.id === inForceId}
                        onSelect={onSelect}
                    />
                ))}
                {hasMore ? (
                    <TableRow className="py-1" style={{ height: "auto" }}>
                        <div role="cell" className="col-span-full flex justify-center">
                            <Button size="sm" busy={loadingMore} onClick={onLoadMore}>
                                {copy.app.loadMore}
                            </Button>
                        </div>
                    </TableRow>
                ) : null}
            </DenseTable>
        </div>
    );
}
