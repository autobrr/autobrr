/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type AlertKind = "APP_UPDATE" | "IRC_UNHEALTHY" | "LIST_REFRESH_ERROR";

type AlertSeverity = "INFO" | "ERROR";

interface Alert {
  kind: AlertKind;
  severity: AlertSeverity;
  subject_id?: number;
  subject: string;
  message?: string;
  url?: string;
  since?: string;
}
