import { QueryClient, QueryClientProvider } from "react-query";
import {BrowserRouter as Router, Route, Redirect} from "react-router-dom";
import { ReactQueryDevtools } from "react-query/devtools";
import { Toaster } from "react-hot-toast";

import Base from "./screens/Base";
import Login from "./screens/auth/login";
import Logout from "./screens/auth/logout";
import { baseUrl } from "./utils";

import { AuthContext, SettingsContext } from "./utils/Context";
import {Fragment, useEffect} from "react";
import {APIClient} from "./api/APIClient";
import {useCookies} from "react-cookie";

function Protected() {
    const [authContext, setAuthContext] = AuthContext.use();
    const [,, removeCookie] = useCookies(['user_session']);

    useEffect(() => {
        APIClient.auth.validate()
            .catch(() => {
                removeCookie("user_session");
                setAuthContext({ username: "", isLoggedIn: false });
            })
    }, [])

    return (
        <Fragment>
            <Toaster position="top-right" />
            {authContext.isLoggedIn ? (
                <Base />
            ) :
                <Redirect to={"/login"}/>
            }
        </Fragment>
    )
}

export const queryClient = new QueryClient();

export function App() {
    const settings = SettingsContext.useValue();

    return (
        <QueryClientProvider client={queryClient}>
            <Router basename={baseUrl()}>
                <Route exact path="/*" component={Protected} />
                <Route exact path="/login" component={Login} />
                <Route exact path="/logout" component={Logout} />
            </Router>
            {settings.debug ? (
                <ReactQueryDevtools initialIsOpen={false} />
            ) : null}
        </QueryClientProvider>
    );
}