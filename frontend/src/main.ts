import { Health } from "./lib/api.js";
import { parseError } from "./lib/errors.js";

const versionElement = document.getElementById("version") as HTMLElement;
const commitElement = document.getElementById("commit") as HTMLElement;
const builtElement = document.getElementById("built") as HTMLElement;
const statusElement = document.getElementById("status") as HTMLElement;

async function load(): Promise<void> {
    const build = await Health.Ping({});
    versionElement.innerText = build.version;
    commitElement.innerText = build.commit;
    builtElement.innerText = build.buildDate;
    statusElement.innerText = "ready";
    statusElement.classList.add("is-ready");
}

load().catch((thrown: unknown) => {
    const failure = parseError(thrown);
    statusElement.innerText = `${failure.code}: ${failure.message}`;
    statusElement.classList.add("is-failed");
});
