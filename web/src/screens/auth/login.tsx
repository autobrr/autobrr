import { useMutation } from "react-query";
import APIClient from "../../api/APIClient";
import { Form, Formik } from "formik";
import { useRecoilState } from "recoil";
import { isLoggedIn } from "../../state/state";
import { useHistory } from "react-router-dom";
import { useEffect } from "react";
import logo from "../../logo.png"
import { TextField, PasswordField } from "../../components/inputs/fields";

interface loginData {
    username: string;
    password: string;
}

function Login() {
    const [loggedIn, setLoggedIn] = useRecoilState(isLoggedIn);
    let history = useHistory();

    useEffect(() => {
        if (loggedIn) {
            // setLoading(false);
            history.push('/');
        } else {
            // setLoading(false);
        }
    }, [loggedIn, history])

    const mutation = useMutation((data: loginData) => APIClient.auth.login(data.username, data.password), {
        onSuccess: () => {
            setLoggedIn(true);
        },
    })

    const handleSubmit = (data: any) => {
        mutation.mutate(data)
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
            <div className="sm:mx-auto sm:w-full sm:max-w-md mb-6">
                <img
                    className="mx-auto h-12 w-auto"
                    src={logo}
                    alt="logo"
                />
            </div>

            <div className="sm:mx-auto sm:w-full sm:max-w-md">
                <div className="bg-white dark:bg-gray-800 py-8 px-4 shadow sm:rounded-lg sm:px-10">

                    <Formik
                        initialValues={{
                            username: "",
                            password: "",
                        }}
                        onSubmit={handleSubmit}
                    >
                        {() => (
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
                        )}
                    </Formik>
                </div>
            </div>
        </div>
    )
}

export default Login;