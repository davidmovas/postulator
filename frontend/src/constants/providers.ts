export const PROVIDER_TYPES = {
    openai: {
        label: "OpenAI",
        description: "GPT model from OpenAI",
        themeColor: "#74AA9C",
        icon: "/openai.svg"
    },
    anthropic: {
        label: "Anthropic",
        description: "Claude model from Anthropic",
        themeColor: "#C4603F",
        icon: "/anthropic.svg"
    },
    google: {
        label: "Google",
        description: "Gemini model from Google",
        themeColor: "#8A6CF5",
        icon: "/google.svg"
    }
} as const;

export type ProviderType = keyof typeof PROVIDER_TYPES;