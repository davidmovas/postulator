export const errorMessages = {
    NOT_FOUND: "That record no longer exists here. The view has been refreshed.",
    CONFLICT: "Something else changed this first. The view has been refreshed.",
    INVALID: "Check the highlighted fields and try again.",
    UNAUTHORIZED: "WordPress refused the credentials for this site.",
    RATE_LIMITED: "The model provider is rate limiting. Retrying shortly.",
    BUDGET_EXCEEDED: "The run stopped at its budget cap. Raise the cap or send fewer pages.",
    EXTERNAL: "An external service failed. Nothing here was changed.",
    INTERNAL: "Something went wrong inside Postulator. The details are in errors.log.",
    CANCELLED: "Cancelled.",
    NEEDS_HUMAN: "This needs your decision before it can continue.",
    LOCKED: "Postulator is locked. Unlock it to continue.",
} as const;
