"use client";

import React, { createContext, useContext, useState } from "react";
import { Site } from "@/models/sites";

interface ModalContextType {
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
}

const ModalContext = createContext<ModalContextType | undefined>(undefined);

export function ModalProvider({ children }: { children: React.ReactNode }) {
    const [createSiteOpen, setCreateSiteOpen] = useState(false);
    const [editSiteOpen, setEditSiteOpen] = useState(false);
    const [passwordOpen, setPasswordOpen] = useState(false);
    const [selectedSite, setSelectedSite] = useState<Site | null>(null);

    const value: ModalContextType = {
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