import { Agent } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Conversation, Message, PendingAction } from "../types.js";

export const createConversation = one(Agent.CreateConversation);
export const setConversationMode = one(Agent.SetMode);
export const sendMessage = one(Agent.Send);
export const confirmAction = one(Agent.Confirm);
export const cancelTurn = one(Agent.Cancel);
export const listConversations = listed<Parameters<typeof Agent.ListConversations>[0], Conversation>(
    Agent.ListConversations,
);
export const listMessages = listed<Parameters<typeof Agent.ListMessages>[0], Message>(Agent.ListMessages);
export const listPendingActions = listed<Parameters<typeof Agent.ListPendingActions>[0], PendingAction>(
    Agent.ListPendingActions,
);
