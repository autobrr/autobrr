/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

/**
 * @vitest-environment node
 */

/// <reference types="node" />

import assert from "node:assert/strict";
import { afterEach, test } from "vitest";

import { CopyTextToClipboard } from "../src/utils/clipboard.ts";

const originalNavigator = Object.getOwnPropertyDescriptor(globalThis, "navigator");
const originalDocument = Object.getOwnPropertyDescriptor(globalThis, "document");
const originalClipboardItem = Object.getOwnPropertyDescriptor(globalThis, "ClipboardItem");

afterEach(() => {
  if (originalNavigator) {
    Object.defineProperty(globalThis, "navigator", originalNavigator);
  } else {
    Reflect.deleteProperty(globalThis, "navigator");
  }

  if (originalDocument) {
    Object.defineProperty(globalThis, "document", originalDocument);
  } else {
    Reflect.deleteProperty(globalThis, "document");
  }

  if (originalClipboardItem) {
    Object.defineProperty(globalThis, "ClipboardItem", originalClipboardItem);
  } else {
    Reflect.deleteProperty(globalThis, "ClipboardItem");
  }
});

interface ClipboardMock {
  write?(items: ClipboardItem[]): Promise<void>;
  writeText?(text: string): Promise<void>;
}

function setNavigator(clipboard?: ClipboardMock) {
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: { clipboard },
  });
}

function setDocument(copyResult: boolean) {
  let appended = false;
  let selected = false;
  let selectionRange: [number, number] | undefined;

  const textarea = {
    focus() {},
    readOnly: false,
    select() {
      selected = true;
    },
    setSelectionRange(start: number, end: number) {
      selectionRange = [start, end];
    },
    style: {},
    value: "",
  };

  const documentMock = {
    activeElement: null,
    body: {
      appendChild(node: unknown) {
        assert.equal(node, textarea);
        appended = true;
      },
      removeChild(node: unknown) {
        assert.equal(node, textarea);
        appended = false;
      },
    },
    createElement(tagName: string) {
      assert.equal(tagName, "textarea");
      return textarea;
    },
    execCommand(command: string) {
      assert.equal(command, "copy");
      assert.equal(appended, true);
      assert.equal(selected, true);
      return copyResult;
    },
  };

  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: documentMock,
  });

  return {
    get appended() {
      return appended;
    },
    get selectionRange() {
      return selectionRange;
    },
    textarea,
  };
}

test("uses the Clipboard API when it succeeds", async () => {
  const copied: string[] = [];
  setNavigator({
    async writeText(text) {
      copied.push(text);
    },
  });

  await CopyTextToClipboard("filter JSON");

  assert.deepEqual(copied, ["filter JSON"]);
});

test("starts a promise-backed clipboard write before the text resolves", async () => {
  let resolveText: ((text: string) => void) | undefined;
  const text = new Promise<string>((resolve) => {
    resolveText = resolve;
  });
  let itemData: Record<string, Promise<Blob>> | undefined;
  let writeStarted = false;

  class TestClipboardItem {
    constructor(data: Record<string, Promise<Blob>>) {
      itemData = data;
    }
  }

  Object.defineProperty(globalThis, "ClipboardItem", {
    configurable: true,
    value: TestClipboardItem,
  });
  setNavigator({
    async write() {
      writeStarted = true;
      assert.ok(itemData);
      const blob = await itemData["text/plain"];
      assert.equal(await blob.text(), "async filter JSON");
    },
  });

  const copy = CopyTextToClipboard(text);

  assert.equal(writeStarted, true);
  resolveText?.("async filter JSON");
  await copy;
});

test("uses the selected-text fallback when the Clipboard API is unavailable", async () => {
  setNavigator();
  const fallback = setDocument(true);

  await CopyTextToClipboard("API key");

  assert.equal(fallback.textarea.value, "API key");
  assert.deepEqual(fallback.selectionRange, [0, 7]);
  assert.equal(fallback.appended, false);
});

test("uses the selected-text fallback when the Clipboard API rejects", async () => {
  setNavigator({
    async writeText() {
      throw new Error("Clipboard API unavailable in this context");
    },
  });
  const fallback = setDocument(true);

  await CopyTextToClipboard("List ID");

  assert.equal(fallback.textarea.value, "List ID");
  assert.deepEqual(fallback.selectionRange, [0, 7]);
});

test("rejects when the fallback copy command fails", async () => {
  setNavigator();
  const fallback = setDocument(false);

  await assert.rejects(
    CopyTextToClipboard("not copied"),
    /browser rejected the clipboard copy command/,
  );
  assert.equal(fallback.appended, false);
});
