"use client";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/use-toast";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { RiRefreshLine, RiPlayLine, RiStopLine, RiDeleteBinLine, RiEyeLine, RiRepeatLine } from "@remixicon/react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import type { Site } from "@/types/site";

interface Job {
  id: number;
  type: string;
  site_id: number;
  site_name?: string;
  article_id?: number;
  status: 'pending' | 'running' | 'completed' | 'failed';
  progress: number;
  error_msg?: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}


export default function JobsPanel() {
  const [jobs, setJobs] = React.useState<Job[]>([]);
  const [sites, setSites] = React.useState<Site[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [selectedJob, setSelectedJob] = React.useState<Job | null>(null);
  const [jobDetailsOpen, setJobDetailsOpen] = React.useState(false);
  const [total, setTotal] = React.useState(0);
  const { toast } = useToast();

  // Manual job creation states
  const [createJobOpen, setCreateJobOpen] = React.useState(false);
  const [selectedSiteId, setSelectedSiteId] = React.useState<number | null>(null);
  const [jobStrategy, setJobStrategy] = React.useState<string>('');

  // Load sites for job creation
  React.useEffect(() => {
    async function loadSites() {
      try {
        const svc = await import("@/services/sites");
        const { items } = await svc.getSites(1, 100);
        setSites(items);
      } catch (e) {
        console.warn('Failed to load sites:', e);
      }
    }
    void loadSites();
  }, []);

  // Load jobs
  const loadJobs = React.useCallback(async () => {
    try {
      setLoading(true);
      
      // Mock implementation - replace with actual backend call
      // const response = await GetJobs({ page, limit: pageSize });
      
      // For now, using mock data
      const mockJobs: Job[] = [
        {
          id: 1,
          type: 'manual',
          site_id: 1,
          site_name: 'Example Site',
          status: 'completed',
          progress: 100,
          created_at: new Date(Date.now() - 3600000).toISOString(),
          completed_at: new Date(Date.now() - 3000000).toISOString(),
          article_id: 123
        },
        {
          id: 2,
          type: 'scheduled',
          site_id: 2,
          site_name: 'Another Site',
          status: 'running',
          progress: 45,
          created_at: new Date(Date.now() - 1800000).toISOString(),
          started_at: new Date(Date.now() - 900000).toISOString()
        },
        {
          id: 3,
          type: 'manual',
          site_id: 1,
          site_name: 'Example Site',
          status: 'failed',
          progress: 0,
          error_msg: 'Failed to connect to WordPress site',
          created_at: new Date(Date.now() - 7200000).toISOString(),
          started_at: new Date(Date.now() - 7000000).toISOString(),
          completed_at: new Date(Date.now() - 6900000).toISOString()
        },
        {
          id: 4,
          type: 'scheduled',
          site_id: 3,
          site_name: 'Third Site',
          status: 'pending',
          progress: 0,
          created_at: new Date(Date.now() - 300000).toISOString()
        }
      ];

      setJobs(mockJobs);
      setTotal(mockJobs.length);
      
    } catch (e) {
      toast({
        title: "Failed to load jobs",
        description: e instanceof Error ? e.message : String(e),
        variant: "destructive"
      });
    } finally {
      setLoading(false);
    }
  }, [toast]);

  React.useEffect(() => {
    void loadJobs();
  }, [loadJobs]);

  // Auto-refresh hook for running jobs
  React.useEffect(() => {
    const hasRunningJobs = jobs.some(job => job.status === 'running' || job.status === 'pending');
    
    if (!hasRunningJobs) return;
    
    const interval = setInterval(() => {
      void loadJobs();
    }, 10000); // Refresh every 10 seconds
    
    return () => clearInterval(interval);
  }, [jobs, loadJobs]);

  const createManualJob = async () => {
    if (!selectedSiteId) {
      toast({
        title: "Site required",
        description: "Please select a site for the job",
        variant: "destructive"
      });
      return;
    }

    try {
      setLoading(true);
      
      // Mock implementation - replace with actual backend call
      // const request = {
      //   site_id: selectedSiteId,
      //   strategy: jobStrategy || undefined
      // };
      // await CreatePublishJob(request);

      toast({
        title: "Job created",
        description: "Publishing job has been created and queued"
      });

      setCreateJobOpen(false);
      setSelectedSiteId(null);
      setJobStrategy('');
      
      // Reload jobs
      await loadJobs();
      
    } catch (e) {
      toast({
        title: "Failed to create job",
        description: e instanceof Error ? e.message : String(e),
        variant: "destructive"
      });
    } finally {
      setLoading(false);
    }
  };

  const retryJob = async (jobId: number) => {
    try {
      setLoading(true);
      
      // Mock implementation - in real app, this would retry the job
      toast({
        title: "Job retried",
        description: "Job has been queued for retry"
      });

      // Reload jobs
      await loadJobs();
      
    } catch (e) {
      toast({
        title: "Failed to retry job",
        description: e instanceof Error ? e.message : String(e),
        variant: "destructive"
      });
    } finally {
      setLoading(false);
    }
  };

  const cancelJob = async (jobId: number) => {
    try {
      setLoading(true);
      
      // Mock implementation - in real app, this would cancel the job
      toast({
        title: "Job cancelled",
        description: "Job has been cancelled"
      });

      // Reload jobs
      await loadJobs();
      
    } catch (e) {
      toast({
        title: "Failed to cancel job",
        description: e instanceof Error ? e.message : String(e),
        variant: "destructive"
      });
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return "—";
    try {
      return new Intl.DateTimeFormat("en-GB", { 
        day: "2-digit", 
        month: "2-digit", 
        year: "numeric", 
        hour: "2-digit", 
        minute: "2-digit" 
      }).format(new Date(dateStr));
    } catch {
      return dateStr;
    }
  };

  const getStatusBadge = (status: Job['status']) => {
    switch (status) {
      case 'pending':
        return <Badge variant="secondary">Pending</Badge>;
      case 'running':
        return <Badge variant="default">Running</Badge>;
      case 'completed':
        return <Badge variant="destructive" className="bg-green-600">Completed</Badge>;
      case 'failed':
        return <Badge variant="destructive">Failed</Badge>;
      default:
        return <Badge variant="outline">{status}</Badge>;
    }
  };

  const getDuration = (startedAt?: string, completedAt?: string) => {
    if (!startedAt) return "—";
    const start = new Date(startedAt);
    const end = completedAt ? new Date(completedAt) : new Date();
    const diffMs = end.getTime() - start.getTime();
    const diffSeconds = Math.floor(diffMs / 1000);
    
    if (diffSeconds < 60) return `${diffSeconds}s`;
    const diffMinutes = Math.floor(diffSeconds / 60);
    if (diffMinutes < 60) return `${diffMinutes}m ${diffSeconds % 60}s`;
    const diffHours = Math.floor(diffMinutes / 60);
    return `${diffHours}h ${diffMinutes % 60}m`;
  };

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">Jobs</h1>
        <p className="text-muted-foreground">Monitor and manage article generation and publishing jobs</p>
      </div>

      {/* Toolbar */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-2">
          <Button onClick={() => void loadJobs()} disabled={loading}>
            <RiRefreshLine size={16} />
            Refresh
          </Button>
          
          <Dialog open={createJobOpen} onOpenChange={setCreateJobOpen}>
            <DialogTrigger asChild>
              <Button>
                <RiPlayLine size={16} />
                Create Job
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-md">
              <DialogHeader>
                <DialogTitle>Create Publishing Job</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label>Site</Label>
                  <Select value={selectedSiteId?.toString() || ""} onValueChange={(value) => setSelectedSiteId(Number(value))}>
                    <SelectTrigger>
                      <SelectValue placeholder="Choose a site..." />
                    </SelectTrigger>
                    <SelectContent>
                      {sites.map((site) => (
                        <SelectItem key={site.id} value={site.id.toString()}>
                          {site.name} ({site.url})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label>Topic Selection Strategy (Optional)</Label>
                  <Select value={jobStrategy} onValueChange={setJobStrategy}>
                    <SelectTrigger>
                      <SelectValue placeholder="Use site default..." />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="unique">Unique - Unused topics only</SelectItem>
                      <SelectItem value="round_robin">Round Robin - Cycle through topics</SelectItem>
                      <SelectItem value="random">Random - From site topics</SelectItem>
                      <SelectItem value="random_all">Random All - From all topics</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex justify-end gap-2">
                  <Button variant="outline" onClick={() => setCreateJobOpen(false)}>
                    Cancel
                  </Button>
                  <Button onClick={createManualJob} disabled={!selectedSiteId || loading}>
                    {loading ? "Creating..." : "Create Job"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>

        <div className="text-sm text-muted-foreground">
          {total} total jobs
        </div>
      </div>

      {/* Jobs Table */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Site</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-center">Progress</TableHead>
              <TableHead>Duration</TableHead>
              <TableHead>Created</TableHead>
              <TableHead>Completed</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {jobs.map((job) => (
              <TableRow key={job.id}>
                <TableCell className="font-mono">{job.id}</TableCell>
                <TableCell>
                  <Badge variant="outline" className="capitalize">
                    {job.type}
                  </Badge>
                </TableCell>
                <TableCell>{job.site_name || `Site ${job.site_id}`}</TableCell>
                <TableCell>{getStatusBadge(job.status)}</TableCell>
                <TableCell className="text-center">
                  <div className="flex items-center gap-2">
                    <Progress value={job.progress} className="w-16 h-2" />
                    <span className="text-xs text-muted-foreground w-8">{job.progress}%</span>
                  </div>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {getDuration(job.started_at, job.completed_at)}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(job.created_at)}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(job.completed_at)}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => {
                        setSelectedJob(job);
                        setJobDetailsOpen(true);
                      }}
                    >
                      <RiEyeLine size={16} />
                    </Button>
                    
                    {job.status === 'failed' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => retryJob(job.id)}
                      >
                        <RiRepeatLine size={16} />
                      </Button>
                    )}
                    
                    {job.status === 'running' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => cancelJob(job.id)}
                      >
                        <RiStopLine size={16} />
                      </Button>
                    )}
                    
                    {(job.status === 'completed' || job.status === 'failed') && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          // Delete job logic here
                        }}
                      >
                        <RiDeleteBinLine size={16} />
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
            {jobs.length === 0 && (
              <TableRow>
                <TableCell colSpan={9} className="text-center text-muted-foreground">
                  {loading ? "Loading jobs..." : "No jobs found"}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Job Details Dialog */}
      {selectedJob && (
        <Dialog open={jobDetailsOpen} onOpenChange={setJobDetailsOpen}>
          <DialogContent className="max-w-2xl">
            <DialogHeader>
              <DialogTitle>Job #{selectedJob.id} Details</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-sm font-medium">Type</Label>
                  <p className="text-sm capitalize">{selectedJob.type}</p>
                </div>
                <div>
                  <Label className="text-sm font-medium">Status</Label>
                  <div className="mt-1">{getStatusBadge(selectedJob.status)}</div>
                </div>
                <div>
                  <Label className="text-sm font-medium">Site</Label>
                  <p className="text-sm">{selectedJob.site_name || `Site ${selectedJob.site_id}`}</p>
                </div>
                <div>
                  <Label className="text-sm font-medium">Progress</Label>
                  <div className="flex items-center gap-2 mt-1">
                    <Progress value={selectedJob.progress} className="w-24 h-2" />
                    <span className="text-sm">{selectedJob.progress}%</span>
                  </div>
                </div>
                <div>
                  <Label className="text-sm font-medium">Created</Label>
                  <p className="text-sm">{formatDate(selectedJob.created_at)}</p>
                </div>
                <div>
                  <Label className="text-sm font-medium">Duration</Label>
                  <p className="text-sm">{getDuration(selectedJob.started_at, selectedJob.completed_at)}</p>
                </div>
              </div>
              
              {selectedJob.article_id && (
                <div>
                  <Label className="text-sm font-medium">Generated Article</Label>
                  <p className="text-sm">Article ID: {selectedJob.article_id}</p>
                </div>
              )}
              
              {selectedJob.error_msg && (
                <div>
                  <Label className="text-sm font-medium">Error Message</Label>
                  <div className="mt-1 p-3 bg-red-50 border border-red-200 rounded text-sm text-red-800">
                    {selectedJob.error_msg}
                  </div>
                </div>
              )}
              
              <div className="flex justify-end gap-2">
                {selectedJob.status === 'failed' && (
                  <Button onClick={() => retryJob(selectedJob.id)}>
                    <RiRepeatLine size={16} />
                    Retry Job
                  </Button>
                )}
                <Button variant="outline" onClick={() => setJobDetailsOpen(false)}>
                  Close
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}

      {/* Auto-refresh */}
      {jobs.some(job => job.status === 'running' || job.status === 'pending') && (
        <div className="mt-4 text-center">
          <p className="text-sm text-muted-foreground">
            Auto-refreshing every 10 seconds for running jobs...
          </p>
        </div>
      )}
    </div>
  );
}