import { createContext, useContext, useEffect, useState } from "react";

export type AuthContext = {
  isLoggedIn: boolean;
  username?: string;
  login: (username: string) => void;
  logout: () => void;
};

const AuthContext = createContext<AuthContext | null>(null);

export const localStorageUserKey = "autobrr_user_auth";

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [username, setUsername] = useState<string | undefined>(() => {
    const storedUser = localStorage.getItem(localStorageUserKey);
    return storedUser ? JSON.parse(storedUser).username : null;
  });

  useEffect(() => {
    if (username) {
      localStorage.setItem(
        localStorageUserKey,
        JSON.stringify({ isLoggedIn: true, username }),
      );
    } else {
      localStorage.removeItem(localStorageUserKey);
    }
  }, [username]);

  const login = (username: string) => {
    setUsername(username);
  };

  const logout = () => {
    setUsername(undefined);
  };

  return (
    <AuthContext.Provider
      value={{ isLoggedIn: !!username, username, login, logout }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
