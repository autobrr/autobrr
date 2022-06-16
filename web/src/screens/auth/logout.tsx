import { useEffect } from "react";
import toast from "react-hot-toast";

import { APIClient } from "../../api/APIClient";
import Toast from "../../components/notifications/Toast";
import { AuthContext } from "../../utils/Context";

export const Logout = () => {
  useEffect(
    () => {
      APIClient.auth.logout()
        .then(() => {
          toast.custom((t) => (
            <Toast type="success" body="You have been logged out. Goodbye!" t={t} />
          ));
          AuthContext.reset();
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
