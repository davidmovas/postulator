"use client";

import React, { createContext, useContext, useState } from "react";
import { Site } from "@/models/sites";
import { Category } from "@/models/categories";
import { ConfirmationModalData } from "@/modals/confirm-modal";

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
    const [selectedSite, setSelectedSite] = useState<Site | null>(null);
    const [selectedSiteId, setSelectedSiteId] = useState<number | null>(null);

    // Categories
    const [createCategoryOpen, setCreateCategoryOpen] = useState(false);
    const [editCategoryOpen, setEditCategoryOpen] = useState(false);
    const [deleteCategoryOpen, setDeleteCategoryOpen] = useState(false);
    const [selectedCategory, setSelectedCategory] = useState<Category | null>(null);

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