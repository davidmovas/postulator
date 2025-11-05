interface BreadcrumbOverride {
    pattern: RegExp;
    getTitle: (segment: string, params: any) => string;
    getHref?: (segment: string, params: any) => string;
}

const breadcrumbOverrides: BreadcrumbOverride[] = [
    {
        pattern: /^\/sites\/(\d+)$/,
        getTitle: (segment, params) => {
            // Здесь нужно будет получать имя сайта из контекста или API
            // Пока возвращаем "Site Details", потом доработаем
            return "Site Details";
        }
    },
    {
        // Для страниц статей /sites/123/articles/456
        pattern: /^\/sites\/(\d+)\/articles\/(\d+)$/,
        getTitle: (segment, params) => {
            return "Article Details";
        }
    }
];