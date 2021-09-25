import { useMutation } from "react-query";
import APIClient from "../../api/APIClient";
import { Form } from "react-final-form";
import { PasswordField, TextField } from "../../components/inputs";
import { useRecoilState } from "recoil";
import { isLoggedIn } from "../../state/state";
import { useHistory } from "react-router-dom";
import { useEffect } from "react";
import logo from "../../logo.png"

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

    const onSubmit = (data: any, form: any) => {
        mutation.mutate(data)
        form.reset()
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

                    <Form
                        initialValues={{
                            username: "",
                            password: "",
                        }}
                        onSubmit={onSubmit}
                    >
                        {({ handleSubmit, values }) => {
                            return (
                                <form className="space-y-6" onSubmit={handleSubmit}>
                                    <TextField name="username" label="Username" autoComplete="username" />
                                    <PasswordField name="password" label="password" autoComplete="current-password" />

                                    {/*<div className="flex items-center justify-between">*/}
                                    {/*    <div className="flex items-center">*/}
                                    {/*        <input*/}
                                    {/*            id="remember-me"*/}
                                    {/*            name="remember-me"*/}
                                    {/*            type="checkbox"*/}
                                    {/*            className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"*/}
                                    {/*        />*/}
                                    {/*        <label htmlFor="remember-me" className="ml-2 block text-sm text-gray-900">*/}
                                    {/*            Remember me*/}
                                    {/*        </label>*/}
                                    {/*    </div>*/}

                                    {/*    <div className="text-sm">*/}
                                    {/*        <a href="#" className="font-medium text-indigo-600 hover:text-indigo-500">*/}
                                    {/*            Forgot your password?*/}
                                    {/*        </a>*/}
                                    {/*    </div>*/}
                                    {/*</div>*/}

                                    <div>
                                        <button
                                            type="submit"
                                            className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 dark:bg-blue-600 hover:bg-indigo-700 dark:hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:focus:ring-blue-500"
                                        >
                                            Sign in
                                        </button>
                                    </div>
                                </form>
                            )
                        }}
                    </Form>
                </div>
            </div>
        </div>
    )
}

export default Login;