/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { formatDistanceToNowStrict, formatISO9075 } from "date-fns";

// sleep for x ms
export function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

// get baseUrl sent from server rendered index template
export function baseUrl() {
  let baseUrl = "/";
  if (window.APP.baseUrl) {
    if (window.APP.baseUrl === "{{.BaseUrl}}") {
      baseUrl = "/";
    } else {
      baseUrl = window.APP.baseUrl;
    }
  }
  return baseUrl;
}

// get routerBasePath sent from server rendered index template
// routerBasePath is used for RouterProvider and does not need work with trailing slash
export function routerBasePath() {
  let baseUrl = "";
  if (window.APP.baseUrl) {
    if (window.APP.baseUrl === "{{.BaseUrl}}") {
      baseUrl = "";
    } else {
      baseUrl = window.APP.baseUrl;
    }
  }
  return baseUrl;
}

// get sseBaseUrl for SSE
export function sseBaseUrl() {
  if (process.env.NODE_ENV === "development")
    return "http://localhost:7474/";

  return `${window.location.origin}${baseUrl()}`;
}

export function classNames(...classes: string[]) {
  return classes.filter(Boolean).join(" ");
}

// column widths for inputs etc
export type COL_WIDTHS = 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12;

// simplify date
export function simplifyDate(date?: string) {
  if (typeof(date) === "string" && date !== "0001-01-01T00:00:00Z") {
    return formatISO9075(new Date(date));
  }
  return "n/a";
}

// if empty date show as n/a
export function IsEmptyDate(date?: string) {
  if (typeof(date) === "string" && date !== "0001-01-01T00:00:00Z") {
    return formatDistanceToNowStrict(
      new Date(date),
      { addSuffix: true }
    );
  }
  return "n/a";
}

export function slugify(str: string) {
  return str
    .normalize("NFKD")
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .trim()
    .replace(/[-\s]+/g, "-");
}

// WARNING: This is not a drop in replacement solution and
// it might not work for some edge cases. Test your code!
export const get = <T> (obj: T, path: string|Array<any>, defValue?: string) => {
  // If path is not defined or it has false value
  if (!path)
    return undefined;
  // Check if path is string or array. Regex : ensure that we do not have '.' and brackets.
  // Regex explained: https://regexr.com/58j0k
  const pathArray = Array.isArray(path) ? path : path.match(/([^[.\]])+/g);
  // Find value
  const result = pathArray && pathArray.reduce(
    (prevObj, key) => prevObj && prevObj[key],
    obj
  );
  // If found value is undefined return default value; otherwise return the value
  return result === undefined ? defValue : result;
};

const UNITS = ['byte', 'kilobyte', 'megabyte', 'gigabyte', 'terabyte', 'petabyte']
const BYTES_PER_KB = 1000


/**
 * Format bytes as human-readable text.
 *
 * @param sizeBytes Number of bytes.
 *
 * @return Formatted string.
 */
export function humanFileSize(sizeBytes: number | bigint): string {
  let size = Math.abs(Number(sizeBytes))

  let u = 0
  while (size >= BYTES_PER_KB && u < UNITS.length - 1) {
    size /= BYTES_PER_KB
    ++u
  }

  return new Intl.NumberFormat([], {
    style: 'unit',
    unit: UNITS[u],
    unitDisplay: 'short',
    maximumFractionDigits: 1,
  }).format(size)
}

/**
 * Format hours as human-readable duration.
 * Converts hours to the largest unit that divides evenly (years, months, weeks, days, hours).
 *
 * @param hours Number of hours.
 *
 * @return Formatted string (e.g., "1 day", "2 weeks", "1 year").
 */
export function formatHoursAsDuration(hours: number): string {
  if (hours === 0) return "0 hours";

  // Try to find the largest unit that divides evenly
  if (hours % 8760 === 0) {
    const years = hours / 8760;
    return `${years} ${years === 1 ? "year" : "years"}`;
  }
  if (hours % 720 === 0) {
    const months = hours / 720;
    return `${months} ${months === 1 ? "month" : "months"}`;
  }
  if (hours % 168 === 0) {
    const weeks = hours / 168;
    return `${weeks} ${weeks === 1 ? "week" : "weeks"}`;
  }
  if (hours % 24 === 0) {
    const days = hours / 24;
    return `${days} ${days === 1 ? "day" : "days"}`;
  }
  return `${hours} ${hours === 1 ? "hour" : "hours"}`;
}

export const RandomLinuxIsos = (count: number) => {
  const linuxIsos = [
    "debian-live-12.10.0-amd64-kde.iso",
    "xubuntu-25.04-desktop-amd64.iso",
    "ubuntu-25.04-live-server-amd64.iso",
    "ubuntu-25.04-desktop-amd64.iso",
    "edubuntu-25.04-desktop-amd64.iso",
    "deepin-desktop-community-23.1-amd64.iso",
    "TUXEDO-OS-202504150920.iso",
    "tails-amd64-6.14.2.iso",
    "manjaro-kde-25.0.0-250414-linux612.iso",
    "Fedora-KDE-Desktop-Live-x86_64-42.iso",
    "manjaro-xfce-25.0.0-250414-linux612.iso",
    "manjaro-gnome-25.0.0-250414-linux612.iso",
    "neon-user-20250410-1320.iso",
    "sparkylinux-7.7-x86_64-xfce.iso",
    "Gobo-017.01-x86_64.iso",
    "lite-7.4-64bit.iso",
    "EndeavourOS_Mercury-Neo-2025.03.19.iso",
    "elementary-8.0.1-20250314.iso",
    "debian-12.10.0-amd64-DVD-1.iso",
    "finnix-250.iso",
    "kali-linux-2025.1a-installer-amd64.iso",
    "linuxmint-22.1-cinnamon-64bit.iso",
    "MX-23.5_x64.iso",
    "Solus-Plasma-Release-2025-01-26.iso"
  ];

  const selectedIsos = [];
  const availableIsos = [...linuxIsos];
  const numToSelect = Math.min(count, availableIsos.length);

  for (let i = 0; i < numToSelect; i++) {
    const randomIndex = Math.floor(Math.random() * availableIsos.length);
    selectedIsos.push(availableIsos.splice(randomIndex, 1)[0]);
  }

  return selectedIsos;
};

export const RandomIsoTracker = (count: number) => {
  const fossTorrentSites = [
    "fosstorrents",
    "linuxtracker",
    "distrowatch",
  ];

  const selectedSites = [];
  const availableSites = [...fossTorrentSites];
  const numToSelect = Math.min(count, availableSites.length);

  for (let i = 0; i < numToSelect; i++) {
    const randomIndex = Math.floor(Math.random() * availableSites.length);
    selectedSites.push(availableSites.splice(randomIndex, 1)[0]);
  }

  return selectedSites;
};

export async function CopyTextToClipboard(text: string) {
  if ("clipboard" in navigator) {
     // Safari requires clipboard operations to be directly triggered by a user interaction.
     // Using setTimeout with a delay of 0 ensures the clipboard operation is deferred until
     // after the current call stack has cleared, effectively placing it outside of the
     // immediate execution context of the user interaction event. This workaround allows
     // the clipboard operation to bypass Safari's security restrictions.
     setTimeout(async () => {
       try {
         await navigator.clipboard.writeText(text);
         console.log("Text copied to clipboard successfully.");
       } catch (err) {
         console.error("Copy to clipboard unsuccessful: ", err);
       }
     }, 0);
  } else {
     // fallback for browsers that do not support the Clipboard API
     copyTextToClipboardFallback(text);
  }
 }
 
 function copyTextToClipboardFallback(text: string) {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  document.body.appendChild(textarea);
  textarea.select();
  try {
     document.execCommand('copy');
     console.log("Text copied to clipboard successfully.");
  } catch (err) {
     console.error('Failed to copy text using fallback method: ', err);
  }
  document.body.removeChild(textarea);
 }


export const IsErrorWithMessage = (error: unknown): error is { message: string } => {
  return typeof error === 'object' && error !== null && 'message' in error;
};
