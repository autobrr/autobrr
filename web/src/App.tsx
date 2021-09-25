import { QueryClient, QueryClientProvider } from "react-query";
import { BrowserRouter as Router, Route } from "react-router-dom";
import Login from "./screens/auth/login";
import Logout from "./screens/auth/logout";
import Base from "./screens/Base";
import { ReactQueryDevtools } from "react-query/devtools";
import Layout from "./components/Layout";
import { baseUrl } from "./utils/utils";

function Protected() {
    return (
        <Layout auth={true}>
            <Base />
        </Layout>
    )
}

export const queryClient = new QueryClient()

function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <Router basename={baseUrl()}>
                <Route exact={true} path="/login" component={Login} />
                <Route exact={true} path="/logout" component={Logout} />
                <Route exact={true} path="/*" component={Protected} />
            </Router>
            <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
    )
};

export default App;