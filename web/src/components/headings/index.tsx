/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

interface Props {
  title: string;
  subtitle: string | React.ReactNode;
  className?: string;
}

export const TitleSubtitle = ({ title, subtitle, className }: Props) => (
  <div className={className}>
    <h2 className="text-lg leading-5 font-bold text-gray-900 dark:text-gray-100">{title}</h2>
    <p className="mt-0.5 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
  </div>
);
