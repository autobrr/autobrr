import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";

import { APIClient } from "../../api/APIClient";
import { AuthContext } from "../../utils/Context";
import Toast from "../../components/notifications/Toast";

export const Logout = () => {
  const navigate = useNavigate();
  useEffect(
    () => {
      APIClient.auth.logout()
        .then(() => {
          AuthContext.reset();
          toast.custom((t) => (
            <Toast type="success" body="You have been logged out. Goodbye!" t={t} />
          ));

          // Dirty way to fix URL without triggering a re-render.
          // Ideally, we'd move the logout component to a function.
          setInterval(() => navigate("/", { replace: true }), 250);
        });
    },
    []
  );

  return (
    <div className="min-h-screen flex justify-center items-center">
      {/*<h1 className="font-bold text-7xl">Goodbye!</h1>*/}
    </div>
  );
};
