import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import { Providers } from "./providers.js";
import "./styles.css";

const host = document.getElementById("root");

if (host === null) {
    throw new Error("the application root element is missing from index.html");
}

createRoot(host).render(
    <StrictMode>
        <Providers />
    </StrictMode>,
);
