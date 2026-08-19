/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

function copyTextToClipboardFallback(text: string): void {
  const activeElement = document.activeElement;
  const textarea = document.createElement("textarea");

  textarea.value = text;
  textarea.readOnly = true;
  textarea.style.position = "fixed";
  textarea.style.top = "0";
  textarea.style.left = "0";
  textarea.style.width = "1px";
  textarea.style.height = "1px";
  textarea.style.opacity = "0";

  document.body.appendChild(textarea);

  try {
    textarea.focus();
    textarea.select();
    textarea.setSelectionRange(0, text.length);

    if (!document.execCommand("copy")) {
      throw new Error("The browser rejected the clipboard copy command.");
    }
  } finally {
    document.body.removeChild(textarea);

    if (activeElement && "focus" in activeElement) {
      (activeElement as HTMLElement).focus({ preventScroll: true });
    }
  }
}

export async function CopyTextToClipboard(text: string | PromiseLike<string>): Promise<void> {
  if (typeof text !== "string") {
    const textPromise = Promise.resolve(text);

    if (
      typeof ClipboardItem === "function" &&
      typeof navigator.clipboard?.write === "function"
    ) {
      try {
        const item = new ClipboardItem({
          "text/plain": textPromise.then((value) => new Blob([value], { type: "text/plain" })),
        });

        await navigator.clipboard.write([item]);
        return;
      } catch {
        return CopyTextToClipboard(await textPromise);
      }
    }

    return CopyTextToClipboard(await textPromise);
  }

  if (typeof navigator.clipboard?.writeText === "function") {
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      copyTextToClipboardFallback(text);
      return;
    }
  }

  copyTextToClipboardFallback(text);
}
