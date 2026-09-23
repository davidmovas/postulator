import { Events } from "@wailsio/runtime";

import type { Envelope, EventType } from "../generated/events.js";

export type Handler<T extends EventType> = (envelope: Envelope<T>) => void;

export function on<T extends EventType>(type: T, handler: Handler<T>): () => void {
    return Events.On(type, (event) => {
        handler(event.data as Envelope<T>);
    });
}
