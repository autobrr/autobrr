/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

export const RandomLinuxIsos = (count: number) => {
  const linuxIsos = [
    "ubuntu-20.04.4-lts-focal-fossa-desktop-amd64-secure-boot",
    "debian-11.3.0-bullseye-amd64-DVD-1-with-nonfree-firmware-netinst",
    "fedora-36-workstation-x86_64-live-iso-with-rpmfusion-free-and-nonfree",
    "archlinux-2023.04.01-x86_64-advanced-installation-environment",
    "linuxmint-20.3-uma-cinnamon-64bit-full-multimedia-support-edition",
    "centos-stream-9-x86_64-dvd1-full-install-iso-with-extended-repositories",
    "opensuse-tumbleweed-20230415-DVD-x86_64-full-packaged-desktop-environments",
    "manjaro-kde-21.1.6-210917-linux514-full-hardware-support-edition",
    "elementaryos-6.1-odin-amd64-20230104-iso-with-pantheon-desktop-environment",
    "pop_os-21.10-amd64-nvidia-proprietary-drivers-included-live",
    "kali-linux-2023.2-live-amd64-iso-with-persistent-storage-and-custom-tools",
    "zorin-os-16-pro-ultimate-edition-64-bit-r1-iso-with-windows-app-support",
    "endeavouros-2023.04.15-x86_64-iso-with-offline-installer-and-xfce4",
    "mx-linux-21.2-aarch64-xfce-iso-with-ahs-enabled-kernel-and-snapshot-feature",
    "solus-4.3-budgie-desktop-environment-full-iso-with-software-center",
    "slackware-15.0-install-dvd-iso-with-extended-documentation-and-extras",
    "alpine-standard-3.15.0-x86_64-iso-for-container-and-server-use",
    "gentoo-livecd-amd64-minimal-20230407-stage3-tarball-included",
    "peppermint-11-20210903-amd64-iso-with-hybrid-lxde-xfce-desktop",
    "deepin-20.3-amd64-iso-with-deepin-desktop-environment-and-app-store"
  ];

  return Array.from({ length: count }, () => linuxIsos[Math.floor(Math.random() * linuxIsos.length)]);
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
 
