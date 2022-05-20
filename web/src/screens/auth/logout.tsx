import { useEffect } from "react";
import { useCookies } from "react-cookie";
import { useHistory } from "react-router-dom";

import { APIClient } from "../../api/APIClient";
import { AuthContext } from "../../utils/Context";

export const Logout = () => {
  const history = useHistory();

  const [, setAuthContext] = AuthContext.use();
  const [,, removeCookie] = useCookies(["user_session"]);

  useEffect(
    () => {
      APIClient.auth.logout()
        .then(() => {
          setAuthContext({ username: "", isLoggedIn: false });
          removeCookie("user_session");

          history.push("/login");
        });
    },
    [history, removeCookie, setAuthContext]
  );

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
      <p>Logged out</p>
    </div>
  );
};
