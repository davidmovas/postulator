'use client';

import React, { useMemo, useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { Job } from '@/services/job';
import { ArrowUpDown, Database, RefreshCw } from 'lucide-react';
import { JobRowActions } from '@/components/jobs/JobRowActions';
import { JobsToolbar } from '@/components/jobs/JobsToolbar';

// Sorting types
export type JobSortField = keyof Job | 'schedule';
export type SortDirection = 'asc' | 'desc' | null;

export interface JobsTableProps {
  jobs: Job[];
  isLoading?: boolean;
  onRefresh: () => Promise<void>;
  onRun: (jobId: number) => Promise<void> | void;
  onPause: (jobId: number) => Promise<void> | void;
  onResume: (jobId: number) => Promise<void> | void;
  onEdit: (job: Job) => void;
  onDelete: (jobId: number) => Promise<void>;
  onCreate: () => void;
}

export function JobsTable({
  jobs,
  isLoading = false,
  onRefresh,
  onRun,
  onPause,
  onResume,
  onEdit,
  onDelete,
  onCreate,
}: JobsTableProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<JobSortField | null>(null);
  const [sortDirection, setSortDirection] = useState<SortDirection>(null);

  const [loadingActions, setLoadingActions] = useState<Record<number, boolean>>({});
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Delete confirmation state
  const [deleteTarget, setDeleteTarget] = useState<Job | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const getStatusVariant = (status: string): 'default' | 'secondary' | 'destructive' | 'outline' => {
    switch ((status || '').toLowerCase()) {
      case 'active':
      case 'running':
        return 'default';
      case 'paused':
        return 'secondary';
      case 'error':
      case 'failed':
        return 'destructive';
      default:
        return 'outline';
    }
  };

    const formatDateTime = (value?: string) => {
        if (!value) return '—';
        const d = new Date(value);
        if (isNaN(d.getTime())) return value;

        const date = d.toLocaleDateString('en-US', {
            day: '2-digit',
            month: 'short',
            year: 'numeric'
        });
        const time = d.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit',
            hour12: false
        });

        return `${date} ${time}`;
    };

  const toHHMMFromNumbers = (h?: number, m?: number) => {
    if (h === undefined || m === undefined || h === null || m === null) return '';
    const hh = String(h).padStart(2, '0');
    const mm = String(m).padStart(2, '0');
    return `${hh}:${mm}`;
  };

  const getScheduleText = (j: Job) => {
    const type = (j.scheduleType || '').toLowerCase();
    if (type === 'manual') return 'Manual';
    if (type === 'once') {
      const t = toHHMMFromNumbers(j.scheduleHour, j.scheduleMinute);
      return t ? `Once @ ${t}` : 'Once';
    }
    if (type === 'interval') {
      const val = j.intervalValue;
      const unit = j.intervalUnit || '';
      if (val && unit) {
        const t = toHHMMFromNumbers(j.scheduleHour, j.scheduleMinute);
        return t ? `Every ${val} ${unit} @ ${t}` : `Every ${val} ${unit}`;
      }
      return 'Interval';
    }
    return '—';
  };

  const handleSort = (field: JobSortField) => {
    if (sortField === field) {
      if (sortDirection === 'asc') setSortDirection('desc');
      else if (sortDirection === 'desc') { setSortField(null); setSortDirection(null); }
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const filteredSorted = useMemo(() => {
    let result = [...jobs];
    // Filter
    const q = searchQuery.trim().toLowerCase();
    if (q) {
      result = result.filter(j =>
        j.name.toLowerCase().includes(q) ||
        (j.status || '').toLowerCase().includes(q) ||
        (j.aiModel || '').toLowerCase().includes(q)
      );
    }

    // Sort
    if (sortField && sortDirection) {
      result.sort((a, b) => {
        let aVal: any;
        let bVal: any;
        if (sortField === 'schedule') { aVal = getScheduleText(a); bVal = getScheduleText(b); }
        else { aVal = (a as any)[sortField]; bVal = (b as any)[sortField]; }
        if (typeof aVal === 'string' && typeof bVal === 'string') {
          return sortDirection === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
        }
        if (typeof aVal === 'number' && typeof bVal === 'number') {
          return sortDirection === 'asc' ? aVal - bVal : bVal - aVal;
        }
        return 0;
      });
    }

    return result;
  }, [jobs, searchQuery, sortField, sortDirection]);

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try { await onRefresh(); } finally { setIsRefreshing(false); }
  };

  const requestDelete = (jobId: number) => {
    const target = jobs.find(j => j.id === jobId) || null;
    setDeleteTarget(target);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    const id = deleteTarget.id;
    setIsDeleting(true);
    setLoadingActions(prev => ({ ...prev, [id]: true }));
    try {
      await onDelete(id);
      setDeleteTarget(null);
    } finally {
      setIsDeleting(false);
      setLoadingActions(prev => ({ ...prev, [id]: false }));
    }
  };

  // Sortable header component
  const SortableHeader = ({ field, children }: { field: JobSortField; children: React.ReactNode }) => (
    <TableHead>
      <button className="flex items-center gap-2 hover:text-foreground transition-colors" onClick={() => handleSort(field)}>
        {children}
        <ArrowUpDown className={`h-4 w-4 ${sortField === field ? 'text-foreground' : 'text-muted-foreground/50'}`} />
      </button>
    </TableHead>
  );

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <JobsToolbar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        isLoading={isLoading}
        totalJobs={jobs.length}
        onCreate={onCreate}
        onRefresh={handleRefresh}
        isRefreshing={isRefreshing}
      />

      {/* Results info */}
      {jobs.length > 0 && (
        <div className="text-sm text-muted-foreground">Showing {filteredSorted.length} of {jobs.length} jobs</div>
      )}

      {/* Table */}
      <div className="w-full overflow-x-auto rounded-lg border">
        <Table className="min-w-[900px] text-sm">
          <TableHeader>
            <TableRow className="[&>th]:py-2">
              <SortableHeader field="name">Name</SortableHeader>
              <SortableHeader field="status">Status</SortableHeader>
              <SortableHeader field="schedule">Schedule</SortableHeader>
              <TableHead>Validation</TableHead>
              <SortableHeader field="aiModel">AI Model</SortableHeader>
              <SortableHeader field="lastRunAt">Last Run</SortableHeader>
              <SortableHeader field="nextRunAt">Next Run</SortableHeader>
              <TableHead className="w-[70px] text-right pr-2">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading && jobs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-center py-12">
                  <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2 text-muted-foreground" />
                  <p className="text-muted-foreground">Loading jobs...</p>
                </TableCell>
              </TableRow>
            ) : filteredSorted.length === 0 ? (
              <TableRow>
                <TableCell colSpan={9} className="text-center py-12">
                  <Database className="h-12 w-12 mx-auto mb-4 text-muted-foreground/50" />
                  <h3 className="font-semibold mb-1">No jobs found</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    {searchQuery ? 'Try adjusting your search query' : 'Get started by creating your first job'}
                  </p>
                  {!searchQuery && (
                    <Button onClick={onCreate} size="sm">New Job</Button>
                  )}
                </TableCell>
              </TableRow>
            ) : (
              filteredSorted.map((j) => (
                <TableRow key={j.id} className="[&>td]:py-2">
                  <TableCell className="font-medium">{j.name}</TableCell>
                  <TableCell>
                    <Badge variant={getStatusVariant(j.status)}>{j.status}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">{getScheduleText(j)}</TableCell>
                  <TableCell>
                    <Badge variant={j.requiresValidation ? 'secondary' : 'outline'}>
                      {j.requiresValidation ? 'Requires validation' : 'Auto publish'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">{j.aiModel || '—'}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{formatDateTime(j.lastRunAt)}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{formatDateTime(j.nextRunAt)}</TableCell>
                  <TableCell className="pr-2">
                    <div className="flex justify-end">
                      <JobRowActions
                        job={j}
                        disabled={loadingActions[j.id]}
                        onEdit={onEdit}
                        onRun={async (id) => {
                          setLoadingActions(prev => ({ ...prev, [id]: true }));
                          try { await onRun(id); } finally { setLoadingActions(prev => ({ ...prev, [id]: false })); }
                        }}
                        onPause={async (id) => {
                          setLoadingActions(prev => ({ ...prev, [id]: true }));
                          try { await onPause(id); } finally { setLoadingActions(prev => ({ ...prev, [id]: false })); }
                        }}
                        onResume={async (id) => {
                          setLoadingActions(prev => ({ ...prev, [id]: true }));
                          try { await onResume(id); } finally { setLoadingActions(prev => ({ ...prev, [id]: false })); }
                        }}
                        onRequestDelete={requestDelete}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}
        title="Delete job?"
        description={deleteTarget ? (<span>Are you sure you want to delete <b>{deleteTarget.name}</b>? This action cannot be undone.</span>) : undefined}
        confirmText="Delete"
        cancelText="Cancel"
        variant="destructive"
        onConfirm={confirmDelete}
        loading={isDeleting}
      />
    </div>
  );
}

export default JobsTable;
