import React, {useEffect, useState} from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import reportWebVitals from './reportWebVitals';
import Base from "./screens/Base";

import {BrowserRouter as Router,} from "react-router-dom";
import {ReactQueryDevtools} from 'react-query/devtools'
import {QueryClient, QueryClientProvider} from 'react-query'

import {RecoilRoot, useRecoilState} from 'recoil';
import {configState} from "./state/state";
import APIClient from "./api/APIClient";
import {APP} from "./domain/interfaces";

declare global {
    interface Window { APP: APP; }
}

window.APP = window.APP || {};

export const queryClient = new QueryClient()

const ConfigWrapper = () => {
    const [config, setConfig] = useRecoilState(configState)
    const [loading, setLoading] = useState(true)

    useEffect(() => {
        APIClient.config.get().then(res => {
                setConfig(res)
                setLoading(false)
            })

    }, [setConfig])

    return (
        <QueryClientProvider client={queryClient}>
            {loading ? null : (
                <Router basename={config.base_url}>
                    <Base/>
                </Router>
            )}
            <ReactQueryDevtools initialIsOpen={false}/>
        </QueryClientProvider>
    )
};


ReactDOM.render(
    <React.StrictMode>
        <RecoilRoot>
            <ConfigWrapper/>
        </RecoilRoot>
    </React.StrictMode>,
    document.getElementById('root')
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
