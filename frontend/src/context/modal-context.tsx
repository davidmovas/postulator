"use client";

import React, { createContext, useContext, useState } from "react";
import { Site } from "@/models/sites";
import { Category } from "@/models/categories";
import { ConfirmationModalData } from "@/modals/confirm-modal";
import { Provider } from "@/models/providers";

interface ModalContextType {
    // Sites
    createSiteModal: {
        isOpen: boolean;
        open: () => void;
        close: () => void;
    };
    editSiteModal: {
        isOpen: boolean;
        open: (site: Site) => void;
        close: () => void;
        site: Site | null;
    };
    passwordModal: {
        isOpen: boolean;
        open: (site: Site) => void;
        close: () => void;
        site: Site | null;
    };
    deleteSiteModal: {
        isOpen: boolean;
        open: (site: Site) => void;
        close: () => void;
        site: Site | null;
    };

    // Categories
    createCategoryModal: {
        isOpen: boolean;
        open: (siteId: number) => void;
        close: () => void;
        siteId: number | null;
    };
    editCategoryModal: {
        isOpen: boolean;
        open: (category: Category) => void;
        close: () => void;
        category: Category | null;
    };
    deleteCategoryModal: {
        isOpen: boolean;
        open: (category: Category) => void;
        close: () => void;
        category: Category | null;
    };

    // Topics
    editTopicModal: {
        isOpen: boolean;
        open: (topic: import("@/models/topics").Topic) => void;
        close: () => void;
        topic: import("@/models/topics").Topic | null;
    };

    // AI-Providers
    createProviderModal: {
        isOpen: boolean;
        open: () => void;
        close: () => void;
    };
    editProviderModal: {
        isOpen: boolean;
        open: (provider: Provider) => void;
        close: () => void;
        provider: Provider | null;
    };
    deleteProviderModal: {
        isOpen: boolean;
        open: (provider: Provider) => void;
        close: () => void;
        provider: Provider | null;
    };

    // Common
    confirmationModal: {
        isOpen: boolean;
        open: (data: ConfirmationModalData) => void;
        close: () => void;
        data: ConfirmationModalData | null;
    };
}

const ModalContext = createContext<ModalContextType | undefined>(undefined);

export function ModalProvider({ children }: { children: React.ReactNode }) {
    // Sites
    const [createSiteOpen, setCreateSiteOpen] = useState(false);
    const [editSiteOpen, setEditSiteOpen] = useState(false);
    const [passwordOpen, setPasswordOpen] = useState(false);
    const [deleteSiteOpen, setDeleteSiteOpen] = useState(false);
    const [selectedSite, setSelectedSite] = useState<Site | null>(null);
    const [selectedSiteId, setSelectedSiteId] = useState<number | null>(null);

    // Categories
    const [createCategoryOpen, setCreateCategoryOpen] = useState(false);
    const [editCategoryOpen, setEditCategoryOpen] = useState(false);
    const [deleteCategoryOpen, setDeleteCategoryOpen] = useState(false);
    const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);

    // Topics
    const [editTopicOpen, setEditTopicOpen] = useState(false);
    const [selectedTopic, setSelectedTopic] = useState<import("@/models/topics").Topic | null>(null);

    // AI-Providers
    const [createProviderOpen, setCreateProviderOpen] = useState(false);
    const [editProviderOpen, setEditProviderOpen] = useState(false);
    const [deleteProviderOpen, setDeleteProviderOpen] = useState(false);
    const [selectedProvider, setSelectedProvider] = useState<Provider | null>(null);

    // Common
    const [confirmationOpen, setConfirmationOpen] = useState(false);
    const [confirmationData, setConfirmationData] = useState<ConfirmationModalData | null>(null);

    const value: ModalContextType = {
        // Sites
        createSiteModal: {
            isOpen: createSiteOpen,
            open: () => setCreateSiteOpen(true),
            close: () => setCreateSiteOpen(false)
        },
        editSiteModal: {
            isOpen: editSiteOpen,
            open: (site) => {
                setSelectedSite(site);
                setEditSiteOpen(true);
            },
            close: () => {
                setEditSiteOpen(false);
                setSelectedSite(null);
            },
            site: selectedSite
        },
        passwordModal: {
            isOpen: passwordOpen,
            open: (site) => {
                setSelectedSite(site);
                setPasswordOpen(true);
            },
            close: () => {
                setPasswordOpen(false);
                setSelectedSite(null);
            },
            site: selectedSite
        },
        deleteSiteModal: {
            isOpen: deleteSiteOpen,
            open: (site) => {
                setSelectedSite(site);
                setDeleteSiteOpen(true);
            },
            close: () => {
                setDeleteSiteOpen(false);
                setSelectedSite(null);
            },
            site: selectedSite
        },

        // Categories
        createCategoryModal: {
            isOpen: createCategoryOpen,
            open: (siteId) => {
                setSelectedSiteId(siteId);
                setCreateCategoryOpen(true);
            },
            close: () => {
                setCreateCategoryOpen(false);
                setSelectedSiteId(null);
            },
            siteId: selectedSiteId
        },
        editCategoryModal: {
            isOpen: editCategoryOpen,
            open: (category) => {
                setSelectedCategory(category);
                setEditCategoryOpen(true);
            },
            close: () => {
                setEditCategoryOpen(false);
                setSelectedCategory(null);
            },
            category: selectedCategory
        },
        deleteCategoryModal: {
            isOpen: deleteCategoryOpen,
            open: (category) => {
                setSelectedCategory(category);
                setDeleteCategoryOpen(true);
            },
            close: () => {
                setDeleteCategoryOpen(false);
                setSelectedCategory(null);
            },
            category: selectedCategory
        },

        // Topics
        editTopicModal: {
            isOpen: editTopicOpen,
            open: (topic) => {
                setSelectedTopic(topic);
                setEditTopicOpen(true);
            },
            close: () => {
                setEditTopicOpen(false);
                setSelectedTopic(null);
            },
            topic: selectedTopic
        },

        // AI-Providers
        createProviderModal: {
            isOpen: createProviderOpen,
            open: () => setCreateProviderOpen(true),
            close: () => setCreateProviderOpen(false)
        },
        editProviderModal: {
            isOpen: editProviderOpen,
            open: (provider) => {
                setSelectedProvider(provider);
                setEditProviderOpen(true);
            },
            close: () => {
                setEditProviderOpen(false);
                setSelectedProvider(null);
            },
            provider: selectedProvider
        },
        deleteProviderModal: {
            isOpen: deleteProviderOpen,
            open: (provider) => {
                setSelectedProvider(provider);
                setDeleteProviderOpen(true);
            },
            close: () => {
                setDeleteProviderOpen(false);
                setSelectedProvider(null);
            },
            provider: selectedProvider
        },

        // Common
        confirmationModal: {
            isOpen: confirmationOpen,
            open: (data) => {
                setConfirmationData(data);
                setConfirmationOpen(true);
            },
            close: () => {
                setConfirmationOpen(false);
                setConfirmationData(null);
            },
            data: confirmationData
        }
    };

    return (
        <ModalContext.Provider value={value}>
            {children}
        </ModalContext.Provider>
    );
}

export function useContextModal() {
    const context = useContext(ModalContext);
    if (context === undefined) {
        throw new Error("useModal must be used within a ModalProvider");
    }
    return context;
}