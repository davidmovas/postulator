import type { ReactElement, SVGProps } from "react";

export type IconProps = Omit<SVGProps<SVGSVGElement>, "width" | "height" | "viewBox" | "fill"> & {
    size?: number;
};

export type IconComponent = ((props: IconProps) => ReactElement) & { path: string };

export const iconViewBox = 960;

export function createIcon(d: string): IconComponent {
    function Icon({ size = 16, ...rest }: IconProps): ReactElement {
        return (
            <svg
                viewBox="0 -960 960 960"
                width={size}
                height={size}
                fill="currentColor"
                focusable="false"
                aria-hidden={rest["aria-label"] === undefined && rest.role === undefined}
                {...rest}
            >
                <path d={d} />
            </svg>
        );
    }
    Icon.path = d;
    return Icon;
}
