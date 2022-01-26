import { newRidgeState } from "react-ridge-state";


export const InitializeGlobalContext = () => {
    const auth_ctx = localStorage.getItem("auth");
    if (auth_ctx)
        AuthContext.set(JSON.parse(auth_ctx));
  
    const settings_ctx = localStorage.getItem("settings");
    if (settings_ctx) {
        SettingsContext.set(JSON.parse(settings_ctx));
    } else {
        // Only check for light theme, otherwise dark theme is the default
        const userMedia = window.matchMedia("(prefers-color-scheme: light)");
        if (userMedia.matches) {
            SettingsContext.set((state) => ({
                ...state,
                darkTheme: false
            }));
        }
    }
}

interface AuthInfo {
    username: string;
    isLoggedIn: boolean;
}

export const AuthContext = newRidgeState<AuthInfo>(
    {
        username: "",
        isLoggedIn: false
    },
    {
        onSet: (new_state) => {
            try {
              localStorage.setItem("auth", JSON.stringify(new_state));
            } catch (e) {
              console.log("An error occurred while trying to modify the local auth context state.");
              console.log("Error:", e);
            }
        }
    }
);

interface SettingsType {
    debug: boolean;
    darkTheme: boolean;
}

export const SettingsContext = newRidgeState<SettingsType>(
  {
      debug: false,
      darkTheme: true
  },
  {
    onSet: (new_state) => {
      try {
        if (new_state.darkTheme) {
            document.documentElement.classList.add("dark");
        } else {
            document.documentElement.classList.remove("dark");
        }
        localStorage.setItem("settings", JSON.stringify(new_state));
      } catch (e) {
        console.log("An error occurred while trying to modify the local settings context state.");
        console.log("Error:", e);
      }
    }
  }
);
