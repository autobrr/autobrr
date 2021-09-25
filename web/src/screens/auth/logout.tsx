import APIClient from "../../api/APIClient";
import {useRecoilState} from "recoil";
import {isLoggedIn} from "../../state/state";
import {useEffect} from "react";
import {useCookies} from "react-cookie";
import {useHistory} from "react-router-dom";

function Logout() {
    const [loggedIn, setLoggedIn] = useRecoilState(isLoggedIn);
    let history = useHistory();

    const [,, removeCookie] = useCookies(['user_session']);

    useEffect(() => {
        APIClient.auth.logout().then(r => {
            removeCookie("user_session")
            setLoggedIn(false);
            history.push('/login');
        })
    }, [loggedIn, history, removeCookie, setLoggedIn])

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-800 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
            <p>Logged out</p>
        </div>
    )
}

export default Logout;