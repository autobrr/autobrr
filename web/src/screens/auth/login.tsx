import { useEffect } from "react";
import { useHistory } from "react-router-dom";
import { useMutation } from "react-query";
import { Form, Formik } from "formik";

import { APIClient } from "../../api/APIClient";
import { TextField, PasswordField } from "../../components/inputs";

import logo from "../../logo.png";
import { AuthContext } from "../../utils/Context";

interface LoginData {
  username: string;
  password: string;
}

export const Login = () => {
  const history = useHistory();
  const [, setAuthContext] = AuthContext.use();

  useEffect(() => {
    // Check if onboarding is available for this instance
    // and redirect if needed
    APIClient.auth.canOnboard()
      .then(() => history.push("/onboard"));
  }, [history]);

  const mutation = useMutation(
    (data: LoginData) => APIClient.auth.login(data.username, data.password),
    {
      onSuccess: (_, variables: LoginData) => {
        setAuthContext({
          username: variables.username,
          isLoggedIn: true
        });
        history.push("/");
      }
    }
  );

  const handleSubmit = (data: LoginData) => mutation.mutate(data);

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md mb-6">
        <img
          className="mx-auto h-12 w-auto"
          src={logo}
          alt="logo"
        />
      </div>
      <div className="sm:mx-auto sm:w-full sm:max-w-md shadow-lg">
        <div className="bg-white dark:bg-gray-800 py-8 px-4 sm:rounded-lg sm:px-10">

          <Formik
            initialValues={{ username: "", password: "" }}
            onSubmit={handleSubmit}
          >
            <Form>
              <div className="space-y-6">
                <TextField name="username" label="Username" columns={6} autoComplete="username" />
                <PasswordField name="password" label="Password" columns={6} autoComplete="current-password" />
              </div>
              <div className="mt-6">
                <button
                  type="submit"
                  className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                >
                  Sign in
                </button>
              </div>
            </Form>
          </Formik>
        </div>
      </div>
    </div>
  );
};
