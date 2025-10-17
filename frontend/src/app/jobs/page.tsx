'use client';

import React, { useEffect, useState } from 'react';
import { JobsTable } from '@/components/jobs/JobsTable';
import { Job, deleteJob, executeJobManually, listJobs, pauseJob, resumeJob } from '@/services/job';
import { useErrorHandling } from '@/lib/error-handling';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { CreateEditJobModal } from '@/components/jobs/CreateEditJobModal';

export default function JobsPage() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const { withErrorHandling } = useErrorHandling();

  // Create/Edit modal state
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<Job | null>(null);

  // Manual run confirmation
  const [runTarget, setRunTarget] = useState<Job | null>(null);
  const [running, setRunning] = useState(false);

  const loadJobs = async () => {
    setIsLoading(true);
    try {
      const data = await listJobs();
      setJobs(data);
      return data;
    } finally {
      setIsLoading(false);
    }
  };

  const handleRefresh = async () => {
    await withErrorHandling(async () => {
      const data = await listJobs();
      setJobs(data);
    }, { successMessage: 'Jobs updated', showSuccess: true });
  };

  const requestRun = (jobId: number) => {
    const target = jobs.find(j => j.id === jobId) || null;
    setRunTarget(target);
  };

  const confirmRun = async () => {
    if (!runTarget) return;
    setRunning(true);
    try {
      await withErrorHandling(async () => {
        await executeJobManually(runTarget.id);
        await loadJobs();
      }, { successMessage: 'Job execution started', showSuccess: true });
      setRunTarget(null);
    } finally {
      setRunning(false);
    }
  };

  const handlePause = async (jobId: number) => {
    await withErrorHandling(async () => {
      await pauseJob(jobId);
      await loadJobs();
    }, { successMessage: 'Job paused', showSuccess: true });
  };

  const handleResume = async (jobId: number) => {
    await withErrorHandling(async () => {
      await resumeJob(jobId);
      await loadJobs();
    }, { successMessage: 'Job resumed', showSuccess: true });
  };

  const handleDelete = async (jobId: number) => {
    await withErrorHandling(async () => {
      await deleteJob(jobId);
      setJobs(prev => prev.filter(j => j.id !== jobId));
    }, { successMessage: 'Job deleted', showSuccess: true });
  };


  const handleEdit = (job: Job) => {
    setEditing(job);
    setEditOpen(true);
  };

  const handleCreate = () => {
    setEditing(null);
    setEditOpen(true);
  };

  useEffect(() => { loadJobs(); }, []);

  return (
    <div className="p-4 md:p-6 lg:p-8 space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Jobs</h1>
        <p className="mt-2 text-muted-foreground">Manage scheduled generation jobs: run, pause, edit and delete.</p>
      </div>

      <JobsTable
        jobs={jobs}
        isLoading={isLoading}
        onRefresh={handleRefresh}
        onRun={requestRun}
        onPause={handlePause}
        onResume={handleResume}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onCreate={handleCreate}
      />

      <CreateEditJobModal
        open={editOpen}
        onOpenChange={setEditOpen}
        job={editing}
        onSaved={async () => { await loadJobs(); }}
      />

      <ConfirmDialog
        open={!!runTarget}
        onOpenChange={(o) => { if (!o) setRunTarget(null); }}
        title="Run job now?"
        description={runTarget ? (<span>Do you want to start execution for <b>{runTarget.name}</b> now?</span>) : undefined}
        confirmText="Run"
        onConfirm={confirmRun}
        loading={running}
      />
    </div>
  );
}
