import { Health } from "../../lib/api.js";
import { one } from "../call.js";

export const ping = one(Health.Ping);
