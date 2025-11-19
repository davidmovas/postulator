const JOB_NAME_PATTERNS = [
    "Job-{site}-#{id}",
];

export function generateJobName(
    siteName?: string,
    existingJobNames: string[] = []
): string {
    const cleanSiteName = (siteName || "Website")
        .replace(/[^\w\s-]/g, '')
        .replace(/[\s_-]+/g, '-')
        .toLowerCase()
        .trim();

    const pattern = JOB_NAME_PATTERNS[Math.floor(Math.random() * JOB_NAME_PATTERNS.length)];

    const jobId = generateUniqueJobId(existingJobNames);

    return pattern
        .replace("{site}", cleanSiteName)
        .replace("{id}", jobId);
}

function generateUniqueJobId(existingJobNames: string[]): string {
    const existingIds = existingJobNames.map(name => {
        const match = name.match(/-(\d{4})$/);
        return match ? parseInt(match[1]) : null;
    }).filter((id): id is number => id !== null);

    let newId: number;
    let attempts = 0;

    do {
        newId = Math.floor(Math.random() * 9000);
        attempts++;

        if (attempts > 10) {
            return Date.now().toString().slice(-4);
        }
    } while (existingIds.includes(newId));

    return newId.toString();
}

export function generateSimpleJobName(siteName?: string): string {
    const cleanSiteName = (siteName || "Website")
        .replace(/[^\w\s-]/g, '')
        .replace(/[\s_-]+/g, '-')
        .toLowerCase()
        .trim();

    const pattern = JOB_NAME_PATTERNS[Math.floor(Math.random() * JOB_NAME_PATTERNS.length)];
    const timestamp = Date.now().toString().slice(-4);

    return pattern
        .replace("{site}", cleanSiteName)
        .replace("{id}", timestamp);
}