/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

// An explicit logout must not land on a login page that starts a new OIDC login,
// or an active provider session signs the user straight back in. Resetting
// AuthContext is what navigates to /login, so the intent travels beside it.
let loggedOut = false;

export const markLoggedOut = () => {
  loggedOut = true;
};

export const consumeLoggedOut = (): boolean => {
  const was = loggedOut;
  loggedOut = false;

  return was;
};

const autoRedirectKey = "autobrr.oidcAutoRedirectAt";
const autoRedirectWindowMs = 30 * 1000;

// Returning to /login without a session soon after an automatic redirect means the
// login did not stick (e.g. the session cookie was dropped), and a silent provider
// login would otherwise bounce the tab between autobrr and the provider.
// sessionStorage is per tab and survives the round trip through the provider.
// Browsers that block storage throw on access; without the guard there is no
// automatic login, so the login page renders instead.
export const claimAutoRedirect = (): boolean => {
  const now = Date.now();

  try {
    const last = Number(sessionStorage.getItem(autoRedirectKey));
    if (last && now - last < autoRedirectWindowMs) {
      return false;
    }

    sessionStorage.setItem(autoRedirectKey, String(now));
  } catch {
    return false;
  }

  return true;
};
