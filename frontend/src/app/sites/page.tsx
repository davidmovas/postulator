'use client';

import { useEffect, useMemo, useState } from 'react';
import { checkSiteHealth, createSite, deleteSite, listSites, setSitePassword, Site } from "@/services/site";
import { SitesTable } from "@/components/tables/SitesTable";
import { useErrorHandling } from "@/lib/error-handling";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { EditSiteModal } from "@/components/modals/EditSiteModal";

export default function SitesPage() {
    const [sites, setSites] = useState<Site[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    const [isCreateOpen, setIsCreateOpen] = useState(false);
    const [step, setStep] = useState<'create' | 'password'>('create');

    const [isEditOpen, setIsEditOpen] = useState(false);
    const [editingSite, setEditingSite] = useState<Site | null>(null);

    const [name, setName] = useState('');
    const [url, setUrl] = useState('');
    const [wpUsername, setWpUsername] = useState('');
    const [password, setPassword] = useState('');

    const [nameTouched, setNameTouched] = useState(false);
    const [createdSiteId, setCreatedSiteId] = useState<number | null>(null);

    const { withErrorHandling, showSuccess } = useErrorHandling();

    const loadSites = async () => {
        setIsLoading(true);
        try {
            const data = await listSites();
            setSites(data);
            return data;
        } catch (error) {
            console.error('Failed to load sites:', error);
        } finally {
            setIsLoading(false);
        }
    };

    // Refresh handler
    const handleRefresh = async () => {
        await withErrorHandling(
            async () => {
                const data = await listSites();
                setSites(data);
                return data;
            },
            {
                successMessage: 'Sites list updated',
                showSuccess: true,
            }
        );
    };

    // Health check single site
    const handleHealthCheck = async (siteId: number) => {
        await withErrorHandling(
            async () => {
                await checkSiteHealth(siteId);
                await loadSites();
            },
            {
                successMessage: `Site health checked successfully`,
                showSuccess: true,
            }
        );
    };

    // Health check all sites
    const handleHealthCheckAll = async () => {
        await withErrorHandling(
            async () => {
                for (const site of sites) {
                    await checkSiteHealth(site.id);
                }
                await loadSites();
            },
            {
                successMessage: 'All sites checked successfully',
                showSuccess: true,
            }
        );
    };

    useEffect(() => {
        loadSites();
    }, []);

    // Edit handler
    const handleEdit = (site: Site) => {
        setEditingSite(site);
        setIsEditOpen(true);
    };

    // Delete handler
    const handleDelete = async (siteId: number) => {
        await withErrorHandling(
            async () => {
                await deleteSite(siteId);
                setSites((prev) => prev.filter((site) => site.id !== siteId));
            },
            {
                successMessage: 'Site deleted successfully',
                showSuccess: true,
            }
        );
    };

    // Helpers
    const normalizeUrl = (value: string): string => {
        if (!value) return value;
        try {
            const hasProtocol = /^https?:\/\//i.test(value);
            const u = new URL(hasProtocol ? value : `http://${value}`);
            return u.toString().replace(/\/$/, '');
        } catch (e) {
            return value;
        }
    };

    const extractDomain = (value: string): string => {
        try {
            const hasProtocol = /^https?:\/\//i.test(value);
            const u = new URL(hasProtocol ? value : `http://${value}`);
            let host = u.hostname.toLowerCase();
            if (host.startsWith('www.')) host = host.slice(4);
            return host;
        } catch (e) {
            return '';
        }
    };

    // Create modal open
    const handleCreate = () => {
        setIsCreateOpen(true);
        setStep('create');
        setName('');
        setUrl('');
        setWpUsername('');
        setPassword('');
        setNameTouched(false);
        setCreatedSiteId(null);
    };

    // Auto-fill name when typing URL
    const handleUrlChange = (value: string) => {
        setUrl(value);
        if (!nameTouched && !name.trim()) {
            const domain = extractDomain(value);
            if (domain) setName(domain);
        }
    };

    const isCreateDisabled = useMemo(() => {
        return !url.trim() || !wpUsername.trim();
    }, [url, wpUsername]);

    const onSubmitCreate = async () => {
        const payload = {
            name: name.trim() || extractDomain(url.trim()) || 'New Site',
            url: normalizeUrl(url.trim()),
            wpUsername: wpUsername.trim(),
        };

        const res = await withErrorHandling(async () => {
            await createSite(payload);
            const all = await loadSites();
            // Find by exact URL match; if multiple, take the one with max id
            const byUrl = (all || []).filter(s => s.url === payload.url);
            let siteId: number | null = null;
            if (byUrl.length > 0) {
                siteId = byUrl.reduce((max, s) => (s.id > max ? s.id : max), byUrl[0].id);
            }
            setCreatedSiteId(siteId);
            return siteId;
        }, { successMessage: 'Site created', showSuccess: true });

        if (res !== null) {
            setStep('password');
        }
    };

    const onSubmitPassword = async () => {
        if (!createdSiteId) {
            setIsCreateOpen(false);
            return;
        }
        const pw = password.trim();
        if (!pw) {
            // allow empty? We'll just skip
            setIsCreateOpen(false);
            return;
        }
        await withErrorHandling(
            async () => {
                await setSitePassword(createdSiteId, pw);
            },
            { successMessage: 'Password set for site', showSuccess: true }
        );
        setIsCreateOpen(false);
    };

    return (
        <div className="p-4 md:p-6 lg:p-8">
            <div className="mb-6">
                <h1 className="text-2xl font-semibold tracking-tight">Sites Management</h1>
                <p className="mt-2 text-muted-foreground">
                    View and manage all your WordPress sites
                </p>
            </div>

            <SitesTable
                sites={sites}
                isLoading={isLoading}
                onRefresh={handleRefresh}
                onHealthCheck={handleHealthCheck}
                onHealthCheckAll={handleHealthCheckAll}
                onEdit={handleEdit}
                onDelete={handleDelete}
                onCreate={handleCreate}
            />

            <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
                <DialogContent>
                    {step === 'create' ? (
                        <>
                            <DialogHeader>
                                <DialogTitle>Add new site</DialogTitle>
                                <DialogDescription>
                                    Enter site details. We’ll auto-fill the name from the URL if left blank. You can edit it anytime.
                                </DialogDescription>
                            </DialogHeader>
                            <div className="space-y-4 pt-2">
                                <div className="space-y-2">
                                    <Label htmlFor="url">URL</Label>
                                    <Input
                                        id="url"
                                        placeholder="https://example.com"
                                        value={url}
                                        onChange={(e) => handleUrlChange(e.target.value)}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label htmlFor="name">Name</Label>
                                    <Input
                                        id="name"
                                        placeholder="example.com"
                                        value={name}
                                        onChange={(e) => { setName(e.target.value); setNameTouched(true); }}
                                    />
                                </div>
                                <div className="space-y-2">
                                    <Label htmlFor="wpuser">User</Label>
                                    <Input
                                        id="wpuser"
                                        placeholder="wordpress username"
                                        value={wpUsername}
                                        onChange={(e) => setWpUsername(e.target.value)}
                                    />
                                </div>
                            </div>
                            <DialogFooter className="pt-4">
                                <Button variant="ghost" onClick={() => setIsCreateOpen(false)}>Cancel</Button>
                                <Button onClick={onSubmitCreate} disabled={isCreateDisabled}>Create</Button>
                            </DialogFooter>
                        </>
                    ) : (
                        <>
                            <DialogHeader>
                                <DialogTitle>Set site password</DialogTitle>
                                <DialogDescription>
                                    Site was created successfully. For security, password is set separately. You can do it now or later.
                                </DialogDescription>
                            </DialogHeader>
                            <div className="space-y-4 pt-2">
                                <div className="space-y-2">
                                    <Label htmlFor="password">Password</Label>
                                    <Input
                                        id="password"
                                        type="password"
                                        placeholder="Enter password"
                                        value={password}
                                        onChange={(e) => setPassword(e.target.value)}
                                    />
                                </div>
                            </div>
                            <DialogFooter className="pt-4">
                                <Button variant="ghost" onClick={() => setIsCreateOpen(false)}>Skip</Button>
                                <Button onClick={onSubmitPassword} disabled={!password.trim() || !createdSiteId}>Set Password</Button>
                            </DialogFooter>
                        </>
                    )}
                </DialogContent>
            </Dialog>

            <EditSiteModal
                open={isEditOpen}
                onOpenChange={setIsEditOpen}
                site={editingSite}
                onSaved={loadSites}
            />
        </div>
    );
}