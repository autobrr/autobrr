import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';

import {APP} from "./domain/interfaces";
import App from "./App";

import { InitializeGlobalContext } from "./utils/Context";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};

// Initializes auth and theme contexts
InitializeGlobalContext();

ReactDOM.render(
    <React.StrictMode>
        <App />
    </React.StrictMode>,
    document.getElementById("root")
);
