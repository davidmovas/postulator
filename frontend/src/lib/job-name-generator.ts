const JOB_NAME_TEMPLATES = [
    "Daily {category} Content",
    "Weekly {topic} Articles",
    "Automated {category} Posts",
    "Scheduled {topic} Publishing",
    "AI-Generated {category} Content",
    "Regular {topic} Updates",
    "{category} Automation Series",
    "Smart {topic} Generator",
    "Continuous {category} Creation",
    "Intelligent {topic} Publisher"
];

const TOPIC_CATEGORY_MAPPING: Record<string, string> = {
    "technology": "Tech",
    "business": "Business",
    "health": "Health",
    "lifestyle": "Lifestyle",
    "entertainment": "Entertainment",
    "sports": "Sports",
    "news": "News",
    "education": "Education"
};

export function generateJobName(
    categoryNames: string[] = [],
    topicTitles: string[] = [],
    promptName?: string
): string {
    const categoryKeywords = extractKeywords(categoryNames);
    const topicKeywords = extractKeywords(topicTitles);

    const primaryKeyword = categoryKeywords[0] || topicKeywords[0] || "Content";

    const template = JOB_NAME_TEMPLATES[Math.floor(Math.random() * JOB_NAME_TEMPLATES.length)];

    let jobName = template.replace("{category}", primaryKeyword)
        .replace("{topic}", primaryKeyword);

    const timestamp = new Date().getTime().toString().slice(-4);
    jobName += ` #${timestamp}`;

    return jobName;
}

function extractKeywords(items: string[]): string[] {
    const allWords = items.flatMap(item =>
        item.toLowerCase()
            .split(/[\s\-\_]+/)
            .filter(word => word.length > 3)
    );

    const frequency: Record<string, number> = {};
    allWords.forEach(word => {
        frequency[word] = (frequency[word] || 0) + 1;
    });

    return Object.entries(frequency)
        .sort(([,a], [,b]) => b - a)
        .map(([word]) => word)
        .slice(0, 5);
}