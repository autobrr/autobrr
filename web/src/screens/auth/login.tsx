import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";
import { useMutation } from "react-query";

import logo from "../../logo.png";
import { APIClient } from "../../api/APIClient";
import { AuthContext } from "../../utils/Context";
import { PasswordInput, TextInput } from "../../components/inputs/text";

type LoginFormFields = {
  username: string;
  password: string;
};

export const Login = () => {
  const { handleSubmit, register, formState } = useForm<LoginFormFields>({
    defaultValues: { username: "", password: "" },
    mode: "onBlur"
  });
  const navigate = useNavigate();
  const [, setAuthContext] = AuthContext.use();

  useEffect(() => {
    // Check if onboarding is available for this instance
    // and redirect if needed
    APIClient.auth.canOnboard()
      .then(() => navigate("/onboard"))
      .catch(() => { /*don't log to console PAHLLEEEASSSE*/ });
  }, []);

  const loginMutation = useMutation(
    (data: LoginFormFields) => APIClient.auth.login(data.username, data.password),
    {
      onSuccess: (_, variables: LoginFormFields) => {
        setAuthContext({
          username: variables.username,
          isLoggedIn: true
        });
        navigate("/");
      }
    }
  );

  const onSubmit = (data: LoginFormFields) => loginMutation.mutate(data);

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
          <form onSubmit={handleSubmit(onSubmit)}>
            <div className="space-y-6">
              <TextInput<LoginFormFields>
                name="username"
                id="username"
                label="username"
                type="text"
                register={register}
                rules={{ required: "Username is required" }}
                errors={formState.errors}
                autoComplete="username"
              />
              <PasswordInput<LoginFormFields>
                name="password"
                id="password"
                label="password"
                register={register}
                rules={{ required: "Password is required" }}
                errors={formState.errors}
                autoComplete="current-password"
              />
            </div>

            <div className="mt-6">
              <button
                type="submit"
                className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 dark:bg-blue-600 hover:bg-blue-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 dark:focus:ring-blue-500"
              >
                Sign in
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
