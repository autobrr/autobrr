import { QueryClient, QueryClientProvider, useQueryErrorResetBoundary } from "react-query";
import { ReactQueryDevtools } from "react-query/devtools";
import { ErrorBoundary } from "react-error-boundary";
import { toast, Toaster } from "react-hot-toast";

import { LocalRouter } from "./domain/routes";
import { AuthContext, SettingsContext } from "./utils/Context";
import { ErrorPage } from "./components/alerts";
import Toast from "./components/notifications/Toast";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { useErrorBoundary: true },
    mutations: {
      onError: (error) => {
        // Use a format string to convert the error object to a proper string without much hassle.
        const message = (
          typeof (error) === "object" && typeof ((error as Error).message) ?
            (error as Error).message :
            `${error}`
        );
        toast.custom((t) => <Toast type="error" body={message} t={t} />);
      }
    }
  }
});

export function App() {
  const { reset } = useQueryErrorResetBoundary();

  const authContext = AuthContext.useValue();
  const settings = SettingsContext.useValue();

  return (
    <QueryClientProvider client={queryClient}>
      <Toaster position="top-right" />
      <ErrorBoundary
        onReset={reset}
        fallbackRender={ErrorPage}
      >
        <LocalRouter isLoggedIn={authContext.isLoggedIn} />
        {settings.debug ? (
          <ReactQueryDevtools initialIsOpen={false} />
        ) : null}
      </ErrorBoundary>
    </QueryClientProvider>
  );
}