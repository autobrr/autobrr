/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface APIKey {
  name: string;
  key: string;
  scopes: string[];
  created_at: Date;
}

type APIKeyAccess = "read" | "write";

interface APIKeyResource {
  name: string;
  access: APIKeyAccess[];
}

interface UserUpdate {
  username_current: string;
  username_new?: string;
  password_current?: string;
  password_new?: string;
}
