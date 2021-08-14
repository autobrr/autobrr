import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';

import {RecoilRoot} from 'recoil';
import {APP} from "./domain/interfaces";
import App from "./App";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};

ReactDOM.render(
    <React.StrictMode>
        <RecoilRoot>
            <App />
        </RecoilRoot>
    </React.StrictMode>,
    document.getElementById('root')
);
