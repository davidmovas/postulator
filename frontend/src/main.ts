import { HealthService } from "../bindings/github.com/davidmovas/postulator/internal/app";

const versionElement = document.getElementById("version") as HTMLElement;
const commitElement = document.getElementById("commit") as HTMLElement;
const builtElement = document.getElementById("built") as HTMLElement;
const statusElement = document.getElementById("status") as HTMLElement;

async function load(): Promise<void> {
    const info = await HealthService.BuildInfo();
    versionElement.innerText = info.version;
    commitElement.innerText = info.commit;
    builtElement.innerText = info.buildDate;
    statusElement.innerText = `ping ${await HealthService.Ping()}`;
    statusElement.classList.add("is-ready");
}

load().catch((err: unknown) => {
    statusElement.innerText = String(err);
    statusElement.classList.add("is-failed");
});
