/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

type ExternalLinkProps = {
  href: string;
  className?: string;
  children?: React.ReactNode;
};

export const ExternalLink = ({ href, className, children }: ExternalLinkProps) => (
  <a
    rel="noopener noreferrer"
    target="_blank"
    href={href}
    className={className}
  >
    {children}
  </a>
);

export const DocsLink = ({ href }: { href: string; }) => (
  <ExternalLink href={href} className="text-blue-700 dark:text-blue-400 visited:text-blue-700 dark:visited:text-blue-400">{href}</ExternalLink>
);
