import type { ReactElement, ReactNode } from "react";
import { createContext, useContext, useMemo } from "react";

import { DenseTable, useWidthOf } from "../../ui/index.js";
import { mapLayout } from "./model/columns.js";
import type { MapColumn, MapLayout } from "./model/columns.js";

const LayoutContext = createContext<MapLayout>(mapLayout(0));

export function useMapColumns(): readonly MapColumn[] {
    return useContext(LayoutContext).columns;
}

export interface MapTableProps {
    label: string;
    children: ReactNode;
}

export function MapTable({ label, children }: MapTableProps): ReactElement {
    const [attach, width] = useWidthOf();
    const layout = useMemo(() => mapLayout(width), [width]);
    return (
        <div ref={attach} data-map-table={layout.columns.join(" ")} className="flex min-h-0 min-w-0 flex-1 flex-col">
            <LayoutContext.Provider value={layout}>
                <DenseTable columns={layout.template} label={label} className="min-h-0 flex-1">
                    {children}
                </DenseTable>
            </LayoutContext.Provider>
        </div>
    );
}
