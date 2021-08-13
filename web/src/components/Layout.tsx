import {isLoggedIn} from "../state/state";
import {useRecoilState} from "recoil";
import {useEffect, useState} from "react";
import { Fragment } from "react";
import {Redirect} from "react-router-dom";
import APIClient from "../api/APIClient";

export default function Layout({auth=false, authFallback="/login", children}: any) {
    const [loggedIn, setLoggedIn] = useRecoilState(isLoggedIn);
    const [loading, setLoading] = useState(auth);

    useEffect(() => {
        // check token
        APIClient.auth.test()
            .then(r => {
                setLoggedIn(true);
                setLoading(false);
            })
            .catch(a => {
                setLoading(false);
            })

    }, [setLoggedIn, loggedIn])

    return (
        <Fragment>
            {loading ? null : (
                <Fragment>
                    {auth && !loggedIn ? <Redirect to={authFallback} /> : (
                        <Fragment>
                            {children}
                        </Fragment>
                    )}
                </Fragment>
            )}
        </Fragment>
    )
}