import { QueryClient, QueryClientProvider } from "react-query";
import { BrowserRouter as Router, Route } from "react-router-dom";
import { ReactQueryDevtools } from "react-query/devtools";
import { Toaster } from "react-hot-toast";

import Base from "./screens/Base";
import Login from "./screens/auth/login";
import Logout from "./screens/auth/logout";
import { baseUrl } from "./utils";

import { AuthContext, SettingsContext } from "./utils/Context";

function Protected() {
    return (
        <>
            <Toaster position="top-right" />
            <Base />
        </>
    )
}

export const queryClient = new QueryClient();

export function App() {
    const authContext = AuthContext.useValue();
    const settings = SettingsContext.useValue();
    return (
        <QueryClientProvider client={queryClient}>
            <Router basename={baseUrl()}>
                {authContext.isLoggedIn ? (
                    <Route exact path="/*" component={Protected} />
                ) : (
                    <Route exact path="/*" component={Login} />
                )}
                <Route exact path="/logout" component={Logout} />
            </Router>
            {settings.debug ? (
                <ReactQueryDevtools initialIsOpen={false} />
            ) : null}
        </QueryClientProvider>
    );
}