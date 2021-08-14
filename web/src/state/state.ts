import { atom } from "recoil";

export const configState = atom({
    key: "configState",
    default: {
        host: "127.0.0.1",
        port: 8989,
        base_url: "",
        log_path: "",
        log_level: "DEBUG",
    }
});

export const isLoggedIn = atom({
    key: 'isLoggedIn',
    default: false,
})