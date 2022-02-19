import { StrictMode } from "react";
import ReactDOM from "react-dom";

import "@fontsource/inter/variable.css";
import "./index.css";

import { App } from "./App";
import { InitializeGlobalContext } from "./utils/Context";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};

// Initializes auth and theme contexts
InitializeGlobalContext();

ReactDOM.render(
    <StrictMode>
        <App />
    </StrictMode>,
    document.getElementById("root")
);
