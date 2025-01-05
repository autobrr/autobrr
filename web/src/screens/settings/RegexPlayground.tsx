/*
 * Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useRef, useState } from "react";

const RegexPlayground = () => {
  const regexRef = useRef<HTMLInputElement>(null);
  const [output, setOutput] = useState<Array<React.ReactElement>>();

  const onInput = (text: string) => {
    if (!regexRef || !regexRef.current)
      return;

    const regexp = new RegExp(regexRef.current.value, "g");

    const results: Array<React.ReactElement> = [];
    text.split("\n").forEach((line, index) => {
      const matches = line.matchAll(regexp);

      let lastIndex = 0;
      for (const match of matches) {
        if (match.index === undefined)
          continue;

        if (!match.length)
          continue;

        const start = match.index;

        let length = 0;
        match.forEach((group) => length += group.length);

        results.push(
          <span key={`match-${start}`}>
            {line.substring(lastIndex, start)}
            <span className="bg-blue-300 text-black font-bold">
              {line.substring(start, start + length)}
            </span>
          </span>
        );
        lastIndex = start + length;
      }

      if (lastIndex < line.length) {
        results.push(
          <span key={`last-${lastIndex + 1}`}>
            {line.substring(lastIndex)}
          </span>
        );
      }

      if (lastIndex > 0)
        results.push(<br key={`line-delim-${index}`} />);
    });

    setOutput(results);
  };

  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700 lg:col-span-9">
      <div className="py-6 px-4 sm:p-6">
        <div>
          <h2 className="text-lg leading-4 font-bold text-gray-900 dark:text-white">Application</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Regex playground. Experiment with your filters here. WIP.
          </p>
        </div>
      </div>
      <div className="px-6 py-4">
        <label
          htmlFor="input-regex"
          className="block text-sm font-medium text-gray-600 dark:text-gray-300"
        >
          RegExp filter
        </label>
        <input
          ref={regexRef}
          id="input-regex"
          type="text"
          autoComplete="true"
          className="mt-1 mb-4 block w-full border rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 border-gray-300 dark:border-gray-700 bg-gray-100 dark:bg-gray-815 dark:text-gray-100 sm:text-sm"
        />
        <label
          htmlFor="input-lines"
          className="block text-sm font-medium text-gray-600 dark:text-gray-300"
        >
          Lines to match
        </label>
        <div
          id="input-lines"
          className="mt-1 mb-4 block w-full dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-md shadow-sm py-2 px-3 focus:outline-none focus:ring-blue-500 focus:border-blue-500 dark:text-gray-100 sm:text-sm"
          onInput={(e) => onInput(e.currentTarget.innerText ?? "")}
          contentEditable
        ></div>
      </div>
      <div className="py-6 px-4 sm:p-6">
        <div>
          <h3 className="text-md leading-6 font-medium text-gray-900 dark:text-white">
            Matches
          </h3>
          <p className="mt-1 text-lg text-gray-500 dark:text-gray-400">
            {output}
          </p>
        </div>
      </div>
    </div>
  );
};

export default RegexPlayground;
