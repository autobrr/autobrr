/*
 * Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { RingResizeSpinner } from "@components/Icons";
import { classNames } from "@utils";

const SIZE = {
  small: "w-6 h-6",
  medium: "w-8 h-8",
  large: "w-12 h-12",
  xlarge: "w-24 h-24"
} as const;

interface SectionLoaderProps {
  $size: keyof typeof SIZE;
}

export const SectionLoader = ({ $size }: SectionLoaderProps) => {
  if ($size === "xlarge") {
    return (
      <div className="max-w-screen-xl mx-auto pb-6 px-4 sm:px-6 lg:pb-16 lg:px-8">
        <RingResizeSpinner className={classNames(SIZE[$size], "mx-auto my-36 text-blue-500")} />
      </div>
    );
  } else {
    return (
      <RingResizeSpinner className={classNames(SIZE[$size], "text-blue-500")} />
    );
  }
};

export function Spinner({ show, wait }: { show?: boolean; wait?: `delay-${number}` }) {
  return (
    <div
      className={`inline-block animate-spin px-3 transition ${
        show ?? true
          ? `opacity-1 duration-500 ${wait ?? 'delay-300'}`
          : 'duration-500 opacity-0 delay-0'
      }`}
    >
      ⍥
    </div>
  )
}
