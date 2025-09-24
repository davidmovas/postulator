"use client";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Checkbox } from "@/components/ui/checkbox";
import { useToast } from "@/components/ui/use-toast";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogDescription, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from "@/components/ui/alert-dialog";
import { RiUploadLine, RiFileTextLine, RiFileExcelLine, RiCodeLine, RiRefreshLine, RiDeleteBinLine, RiExchangeLine } from "@remixicon/react";
import type { Site } from "@/types/site";
import type { SiteTopicLink } from "@/services/siteTopics";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

interface ImportPreviewItem {
  title: string;
  keywords?: string;
  category?: string;
  tags?: string;
  isDuplicate: boolean;
  status: 'new' | 'duplicate' | 'error';
}

export default function SiteTopicsPanel() {
  const [sites, setSites] = React.useState<Site[]>([]);
  const [selectedSiteId, setSelectedSiteId] = React.useState<number | null>(null);
  const [siteTopics, setSiteTopics] = React.useState<SiteTopicLink[]>([]);
  const [selected, setSelected] = React.useState<Set<number>>(new Set());
  const [loading, setLoading] = React.useState(false);
  const { toast } = useToast();

  // Import states
  const [importDialogOpen, setImportDialogOpen] = React.useState(false);
  const [importFormat, setImportFormat] = React.useState<'txt' | 'csv' | 'jsonl'>('txt');
  const [importFile, setImportFile] = React.useState<File | null>(null);
  const [importPreview, setImportPreview] = React.useState<ImportPreviewItem[]>([]);

  // Reassign states
  const [reassignDialogOpen, setReassignDialogOpen] = React.useState(false);
  const [targetSiteId, setTargetSiteId] = React.useState<number | null>(null);

  // Load sites
  React.useEffect(() => {
    async function loadSites() {
      try {
        const svc = await import("@/services/sites");
        const { items } = await svc.getSites(1, 100);
        setSites(items);
        if (items.length > 0 && !selectedSiteId) {
          setSelectedSiteId(items[0].id);
        }
      } catch (e) {
        toast({ 
          title: "Failed to load sites", 
          description: e instanceof Error ? e.message : String(e), 
          variant: "destructive" 
        });
      }
    }
    void loadSites();
  }, [toast, selectedSiteId]);

  // Load site topics when site changes
  React.useEffect(() => {
    async function loadSiteTopics() {
      if (!selectedSiteId) return;
      
      try {
        setLoading(true);
        const svc = await import("@/services/siteTopics");
        const { items } = await svc.getSiteTopics(selectedSiteId, 1, 1000);
        setSiteTopics(items);
      } catch (e) {
        toast({ 
          title: "Failed to load site topics", 
          description: e instanceof Error ? e.message : String(e), 
          variant: "destructive" 
        });
      } finally {
        setLoading(false);
      }
    }
    void loadSiteTopics();
  }, [selectedSiteId, toast]);

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file || !selectedSiteId) return;

    setImportFile(file);
    
    try {
      const fileContent = await file.text();
      
      // Use backend preview functionality
      const { TopicsImport } = await import("@/wailsjs/wailsjs/go/bindings/Binder");
      const { dto } = await import("@/wailsjs/wailsjs/go/models");
      
      // Create preview request
      const request = new dto.TopicsImportRequest({
        site_id: selectedSiteId,
        file_content: fileContent,
        file_format: importFormat,
        preview_only: true
      });

      // Get preview from backend
      const previewResult = await TopicsImport(selectedSiteId, request);
      
      // Convert backend preview to frontend format
      const preview: ImportPreviewItem[] = previewResult.topics?.map((topic: {
        title: string;
        keywords?: string;
        category?: string;
        tags?: string;
        status: string;
      }) => ({
        title: topic.title,
        keywords: topic.keywords,
        category: topic.category,
        tags: topic.tags,
        isDuplicate: topic.status === 'duplicate' || topic.status === 'exists',
        status: topic.status === 'duplicate' || topic.status === 'exists' ? 'duplicate' : 'new'
      })) || [];
      
      setImportPreview(preview);
      
    } catch (e) {
      toast({ 
        title: "Failed to parse file", 
        description: e instanceof Error ? e.message : String(e), 
        variant: "destructive" 
      });
    }
  };


  const executeImport = async () => {
    if (!selectedSiteId || !importFile) return;

    try {
      setLoading(true);
      const fileContent = await importFile.text();
      
      // Import the TopicsImport binding and dto
      const { TopicsImport } = await import("@/wailsjs/wailsjs/go/bindings/Binder");
      const { dto } = await import("@/wailsjs/wailsjs/go/models");
      
      // Create the import request
      const request = new dto.TopicsImportRequest({
        site_id: selectedSiteId,
        file_content: fileContent,
        file_format: importFormat,
        preview_only: false
      });

      // Call the backend import function
      const result = await TopicsImport(selectedSiteId, request);
      
      // Handle the response based on whether it's a preview or actual import
      if (result.created_topics !== undefined) {
        // This is an import result
        toast({ 
          title: "Import completed", 
          description: `Created ${result.created_topics} topics, reused ${result.reused_topics}, skipped ${result.skipped_topics}` 
        });
      } else {
        toast({ 
          title: "Import completed", 
          description: "Topics imported successfully" 
        });
      }

      setImportFile(null);
      setImportPreview([]);
      
      // Reload site topics
      const siteTopicsSvc = await import("@/services/siteTopics");
      const { items } = await siteTopicsSvc.getSiteTopics(selectedSiteId, 1, 1000);
      setSiteTopics(items);
      
    } catch (e) {
      toast({ 
        title: "Import failed", 
        description: e instanceof Error ? e.message : String(e), 
        variant: "destructive" 
      });
    } finally {
      setLoading(false);
    }
  };

  const handleReassign = async () => {
    if (!targetSiteId || selected.size === 0) return;

    try {
      setLoading(true);
      const svc = await import("@/services/siteTopics");

      // This would need a backend method for bulk reassignment
      // For now, we'll simulate it by deleting from current site and adding to target site
      const promises = Array.from(selected).map(async (siteTopicId) => {
        const siteTopicLink = siteTopics.find(st => st.id === siteTopicId);
        if (!siteTopicLink) return;

        // Delete from current site
        await svc.deleteSiteTopic(siteTopicId);
        
        // Add to target site
        await svc.createSiteTopic({
          site_id: targetSiteId,
          topic_id: siteTopicLink.topic_id,
          priority: siteTopicLink.priority
        });
      });

      await Promise.all(promises);

      toast({ 
        title: "Topics reassigned", 
        description: `${selected.size} topics moved to selected site` 
      });

      setReassignDialogOpen(false);
      setSelected(new Set());
      setTargetSiteId(null);
      
      // Reload current site topics
      if (selectedSiteId) {
        const { items } = await svc.getSiteTopics(selectedSiteId, 1, 1000);
        setSiteTopics(items);
      }

    } catch (e) {
      toast({ 
        title: "Reassignment failed", 
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

  const toggleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelected(new Set(siteTopics.map(st => st.id)));
    } else {
      setSelected(new Set());
    }
  };

  const toggleRow = (id: number, checked: boolean) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (checked) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  };

  return (
    <div className="p-4 md:p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-semibold">Site Topics</h1>
        <p className="text-muted-foreground">Import topics, manage assignments, and view usage statistics</p>
      </div>

      {/* Site Selection */}
      <div className="mb-6">
        <Label htmlFor="site-select" className="text-base font-medium">Select Site</Label>
        <Select value={selectedSiteId?.toString() || ""} onValueChange={(value) => setSelectedSiteId(Number(value))}>
          <SelectTrigger className="w-full max-w-md mt-2">
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

      {selectedSiteId && (
        <Tabs defaultValue="topics" className="w-full">
          <TabsList className="mb-4">
            <TabsTrigger value="topics">Topics Table</TabsTrigger>
            <TabsTrigger value="import">Import</TabsTrigger>
          </TabsList>

          <TabsContent value="topics">
            {/* Toolbar */}
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    // Refresh
                    if (selectedSiteId) {
                      const loadSiteTopics = async () => {
                        const svc = await import("@/services/siteTopics");
                        const { items } = await svc.getSiteTopics(selectedSiteId, 1, 1000);
                        setSiteTopics(items);
                      };
                      void loadSiteTopics();
                    }
                  }}
                >
                  <RiRefreshLine size={16} />
                  Refresh
                </Button>
                
                {selected.size > 0 && (
                  <>
                    <Dialog open={reassignDialogOpen} onOpenChange={setReassignDialogOpen}>
                      <DialogTrigger asChild>
                        <Button size="sm" variant="outline">
                          <RiExchangeLine size={16} />
                          Reassign ({selected.size})
                        </Button>
                      </DialogTrigger>
                      <DialogContent>
                        <DialogHeader>
                          <DialogTitle>Reassign {selected.size} Topics</DialogTitle>
                        </DialogHeader>
                        <div className="space-y-4">
                          <div>
                            <Label>Target Site</Label>
                            <Select value={targetSiteId?.toString() || ""} onValueChange={(value) => setTargetSiteId(Number(value))}>
                              <SelectTrigger>
                                <SelectValue placeholder="Choose target site..." />
                              </SelectTrigger>
                              <SelectContent>
                                {sites.filter(s => s.id !== selectedSiteId).map((site) => (
                                  <SelectItem key={site.id} value={site.id.toString()}>
                                    {site.name} ({site.url})
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                          <div className="flex justify-end gap-2">
                            <Button variant="outline" onClick={() => setReassignDialogOpen(false)}>
                              Cancel
                            </Button>
                            <Button onClick={handleReassign} disabled={!targetSiteId || loading}>
                              {loading ? "Processing..." : "Reassign"}
                            </Button>
                          </div>
                        </div>
                      </DialogContent>
                    </Dialog>

                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button size="sm" variant="destructive">
                          <RiDeleteBinLine size={16} />
                          Delete ({selected.size})
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete {selected.size} site topic links?</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will remove the selected topics from this site, but the topics themselves will remain.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            onClick={async () => {
                              try {
                                const svc = await import("@/services/siteTopics");
                                await Promise.all(Array.from(selected).map(id => svc.deleteSiteTopic(id)));
                                setSelected(new Set());
                                
                                if (selectedSiteId) {
                                  const { items } = await svc.getSiteTopics(selectedSiteId, 1, 1000);
                                  setSiteTopics(items);
                                }
                                
                                toast({ title: "Topics removed from site" });
                              } catch (e) {
                                toast({ 
                                  title: "Failed to delete", 
                                  description: e instanceof Error ? e.message : String(e), 
                                  variant: "destructive" 
                                });
                              }
                            }}
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </>
                )}
              </div>
            </div>

            {/* Topics Table */}
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[50px]">
                      <Checkbox 
                        checked={selected.size > 0 && selected.size === siteTopics.length}
                        onCheckedChange={toggleSelectAll}
                      />
                    </TableHead>
                    <TableHead className="w-[25%]">Title</TableHead>
                    <TableHead>Keywords</TableHead>
                    <TableHead>Category</TableHead>
                    <TableHead>Tags</TableHead>
                    <TableHead className="text-center">Usage Count</TableHead>
                    <TableHead>Last Used</TableHead>
                    <TableHead className="text-center">Priority</TableHead>
                    <TableHead className="text-center">Round Robin Pos</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {siteTopics.map((siteTopicLink) => (
                    <TableRow key={siteTopicLink.id} data-state={selected.has(siteTopicLink.id) ? "selected" : undefined}>
                      <TableCell>
                        <Checkbox 
                          checked={selected.has(siteTopicLink.id)}
                          onCheckedChange={(checked) => toggleRow(siteTopicLink.id, Boolean(checked))}
                        />
                      </TableCell>
                      <TableCell className="font-medium">{siteTopicLink.topic_title || "—"}</TableCell>
                      <TableCell className="text-muted-foreground text-sm">—</TableCell>
                      <TableCell className="text-sm">—</TableCell>
                      <TableCell className="text-sm">—</TableCell>
                      <TableCell className="text-center">{siteTopicLink.usage_count}</TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        {formatDate(siteTopicLink.last_used_at)}
                      </TableCell>
                      <TableCell className="text-center">{siteTopicLink.priority}</TableCell>
                      <TableCell className="text-center">{siteTopicLink.round_robin_pos}</TableCell>
                    </TableRow>
                  ))}
                  {siteTopics.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={9} className="text-center text-muted-foreground">
                        {loading ? "Loading..." : "No topics assigned to this site"}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>

          <TabsContent value="import">
            <div className="max-w-2xl">
              <div className="space-y-4">
                <div>
                  <Label className="text-base font-medium">Import Format</Label>
                  <Select value={importFormat} onValueChange={(value) => setImportFormat(value as 'txt' | 'csv' | 'jsonl')}>
                    <SelectTrigger className="w-full mt-2">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="txt">
                        <div className="flex items-center">
                          <RiFileTextLine size={16} className="mr-2" />
                          Text (.txt) - One topic per line
                        </div>
                      </SelectItem>
                      <SelectItem value="csv">
                        <div className="flex items-center">
                          <RiFileExcelLine size={16} className="mr-2" />
                          CSV (.csv) - With headers
                        </div>
                      </SelectItem>
                      <SelectItem value="jsonl">
                        <div className="flex items-center">
                          <RiCodeLine size={16} className="mr-2" />
                          JSON Lines (.jsonl)
                        </div>
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <Label className="text-base font-medium">Upload File</Label>
                  <div className="mt-2 border-2 border-dashed border-muted-foreground/25 rounded-lg p-6 text-center">
                    <RiUploadLine size={48} className="mx-auto mb-4 text-muted-foreground" />
                    <Input 
                      type="file" 
                      accept={importFormat === 'txt' ? '.txt' : importFormat === 'csv' ? '.csv' : '.jsonl'}
                      onChange={handleFileChange}
                      className="mb-2"
                    />
                    <p className="text-sm text-muted-foreground">
                      {importFormat === 'txt' && "Upload a text file with one topic title per line"}
                      {importFormat === 'csv' && "Upload a CSV file with columns: title, keywords, category, tags"}
                      {importFormat === 'jsonl' && "Upload a JSONL file with objects containing title, keywords, category, tags"}
                    </p>
                  </div>
                </div>

                {importPreview.length > 0 && (
                  <div>
                    <h3 className="font-medium mb-2">Preview ({importPreview.length} items)</h3>
                    <div className="max-h-64 overflow-y-auto border rounded">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Title</TableHead>
                            <TableHead>Status</TableHead>
                            {importFormat !== 'txt' && (
                              <>
                                <TableHead>Keywords</TableHead>
                                <TableHead>Category</TableHead>
                                <TableHead>Tags</TableHead>
                              </>
                            )}
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {importPreview.slice(0, 10).map((item, index) => (
                            <TableRow key={index}>
                              <TableCell className="font-medium">{item.title}</TableCell>
                              <TableCell>
                                <span className={`px-2 py-1 text-xs rounded ${
                                  item.isDuplicate ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800'
                                }`}>
                                  {item.isDuplicate ? 'Duplicate' : 'New'}
                                </span>
                              </TableCell>
                              {importFormat !== 'txt' && (
                                <>
                                  <TableCell className="text-sm">{item.keywords || "—"}</TableCell>
                                  <TableCell className="text-sm">{item.category || "—"}</TableCell>
                                  <TableCell className="text-sm">{item.tags || "—"}</TableCell>
                                </>
                              )}
                            </TableRow>
                          ))}
                          {importPreview.length > 10 && (
                            <TableRow>
                              <TableCell colSpan={importFormat === 'txt' ? 2 : 5} className="text-center text-muted-foreground">
                                ... and {importPreview.length - 10} more items
                              </TableCell>
                            </TableRow>
                          )}
                        </TableBody>
                      </Table>
                    </div>

                    <div className="flex items-center justify-between mt-4">
                      <div className="text-sm text-muted-foreground">
                        {importPreview.filter(item => !item.isDuplicate).length} new, {importPreview.filter(item => item.isDuplicate).length} duplicates
                      </div>
                      <Button onClick={executeImport} disabled={loading}>
                        {loading ? "Importing..." : "Import Topics"}
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}