"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ReactNode } from "react";

interface SettingsSectionProps {
    title: string;
    children: ReactNode;
    icon?: ReactNode;
}

export function SettingsSection({ title, children, icon }: SettingsSectionProps) {
    return (
        <Card>
            <CardHeader className="pb-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                        {icon && <div className="text-primary">{icon}</div>}
                        <CardTitle className="text-lg">{title}</CardTitle>
                    </div>
                </div>
            </CardHeader>
            <CardContent>
                {children}
            </CardContent>
        </Card>
    );
}