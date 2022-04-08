import { Fragment } from "react";
import { BrowserRouter as Router, Route, Switch } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "react-query";
import { ReactQueryDevtools } from "react-query/devtools";
import { Toaster } from "react-hot-toast";

import Base from "./screens/Base";
import { Login } from "./screens/auth/login";
import { Logout } from "./screens/auth/logout";
import { Onboarding } from "./screens/auth/onboarding";
import { baseUrl } from "./utils";

import { AuthContext, SettingsContext } from "./utils/Context";

function Protected() {
  return (
    <Fragment>
      <Toaster position="top-right" />
      <Base />
    </Fragment>
  )
}

export const queryClient = new QueryClient();

export function App() {
  const authContext = AuthContext.useValue();
  const settings = SettingsContext.useValue();

  return (
    <QueryClientProvider client={queryClient}>
      <Router basename={baseUrl()}>
        <Route exact path="/logout" component={Logout} />

        {authContext.isLoggedIn ? (
          <Route component={Protected} />
        ) : (
          <Switch>
            <Route exact path="/onboard" component={Onboarding} />
            <Route component={Login} />
          </Switch>
        )}
      </Router>
      {settings.debug ? (
        <ReactQueryDevtools initialIsOpen={false} />
      ) : null}
    </QueryClientProvider>
  );
}