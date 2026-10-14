/*
 * Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { formatDistanceToNowStrict } from "date-fns";

import { simplifyDate } from "@utils";

interface InboxAgeProps {
  createdAt: string;
  className?: string;
}

/** InboxAge renders how long ago a message arrived, with the full date on hover. */
export const InboxAge = ({ createdAt, className }: InboxAgeProps) => (
  <time className={className} dateTime={createdAt} title={simplifyDate(createdAt)}>
    {formatDistanceToNowStrict(new Date(createdAt), { addSuffix: false })}
  </time>
);
