import { Form, Formik } from "formik";
import { useMutation } from "react-query";
import { useNavigate } from "react-router-dom";
import { APIClient } from "../../api/APIClient";

import { TextField, PasswordField } from "../../components/inputs";
import logo from "../../logo.png";

interface InputValues {
  username: string;
  password1: string;
  password2: string;
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

  const navigate = useNavigate();

  const mutation = useMutation(
    (data: InputValues) => APIClient.auth.onboard(data.username, data.password1),
    { onSuccess: () => navigate("/") }
  );

  return (
    <div className="min-h-screen flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md mb-6">
        <img className="mx-auto h-12 w-auto" src={logo} alt="logo"/>
        <h1 className="text-center text-gray-900 dark:text-gray-200 font-bold pt-2 text-2xl">
          autobrr
        </h1>
      </div>
      <div className="sm:mx-auto sm:w-full sm:max-w-md shadow-lg">
        <div className="bg-white dark:bg-gray-800 py-8 px-4 sm:rounded-lg sm:px-10">
          <Formik
            initialValues={{
              username: "",
              password1: "",
              password2: ""
            }}
            onSubmit={(data) => mutation.mutate(data)}
            validate={validate}
          >
            <Form>
              <div className="space-y-6">
                <TextField name="username" label="Username" columns={6} autoComplete="username" />
                <PasswordField name="password1" label="Password" columns={6} autoComplete="current-password" />
                <PasswordField name="password2" label="Confirm password" columns={6} autoComplete="current-password" />
              </div>
              <div className="mt-6">
                <button
                  type="submit"
                  className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
                >
                  Create account
                </button>
              </div>
            </Form>
          </Formik>
        </div>
      </div>
    </div>
  );
};

