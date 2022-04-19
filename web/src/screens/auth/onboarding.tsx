import { Form, Formik } from "formik";
import { useMutation, useQuery } from "react-query";
import { useHistory } from "react-router-dom";
import { toast } from "react-hot-toast";

import { APIClient } from "../../api/APIClient";
import { Toast } from "../../components/notifications/Toast";
import { TextField, PasswordField } from "../../components/inputs";

interface InputValues {
  username: string;
  password1: string;
  password2: string;
  logDir: string;
}

export const Onboarding = () => {
  const validate = (values: InputValues) => {
    const obj: Record<string, string> = {};

    if (!values.username)
      obj.username = "Required";
    
    if (!values.password1)
      obj.password1 = "Required";

    if (!values.password2)
      obj.password2 = "Required";

    if (values.password1 !== values.password2)
      obj.password2 = "Passwords don't match!";
    
    return obj;
  };

  const { data: preferences, isError } = useQuery(
    "onboarding_preferences",
    APIClient.auth.getOnboardingPreferences,
    {
      onSuccess: (responseData) => {
        // Show all errors/warnings encountered while searching for a preferred log directory.
        if (responseData.log_errors === undefined)
          return;
        
        responseData.log_errors.forEach((errString) => {
          toast.custom((t) => <Toast type="error" body={errString} t={t} />);
        });
      }
    }
  );

  const history = useHistory();

  const mutation = useMutation(
    (data: InputValues) => (
      APIClient.auth.onboard(
        data.username,
        data.password1,
        data.logDir
      )
    ),
    {
      onSuccess: () => {
        history.push("/login");
      }
    }
  );

  if (!preferences)
    return null;

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <h1
        className="text-3xl text-center font-bold text-gray-900 dark:text-gray-200 my-4"
      >
        {!isError ? "Create a new user" : "Onboarding is currently unavailable."}
      </h1>
      {preferences && !isError ? (
        <div className="sm:mx-auto sm:w-full sm:max-w-md shadow-lg">
          <div className="bg-white dark:bg-gray-800 py-8 px-4 sm:rounded-lg sm:px-10">
            <Formik
              initialValues={{
                username: "",
                password1: "",
                password2: "",
                logDir: preferences.log_dir
              }}
              onSubmit={(data) => mutation.mutate(data)}
              validate={validate}
            >
              <Form>
                <div className="space-y-3">
                  <TextField name="username" label="Username" columns={6} autoComplete="username" />
                  <PasswordField name="password1" label="Password" columns={6} autoComplete="current-password" />
                  <PasswordField name="password2" label="Confirm password" columns={6} autoComplete="current-password" />
                  <TextField name="logDir" label="Log dir" columns={6} />
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  Leave Log dir empty if you wish not to keep log files.
                </p>
                <div className="mt-4">
                  <button
                    type="submit"
                    className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                  >
                    Create user!
                  </button>
                </div>
              </Form>
            </Formik>
          </div>
        </div>
      ) : null}
    </div>
  );
};

