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
