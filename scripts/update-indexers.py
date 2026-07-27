#!/usr/bin/env python3

# Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
# SPDX-License-Identifier: GPL-2.0-or-later

"""Generate the indexer and freeleech tables for the autobrr.com docs.

Indexer definitions (v2) parse announces per IRC channel: every channel carries
its own set of line patterns, and the release fields come from the named capture
groups in those patterns plus the optional mappings that translate a captured
value into other fields. Freeleech support is therefore derived from the groups
and mappings of every channel of a definition, not from a definition-level vars
list like it was in v1.
"""

import argparse
import os
import re
import sys
from typing import Any, Dict, List, Set, Tuple

import yaml

# Paths relative to GitHub Actions workspace
DEFINITIONS_DIR = "../autobrr/internal/indexer/definitions"
INDEXERS_OUTPUT = "../autobrr.com/snippets/indexers.mdx"
FREELEECH_OUTPUT = "../autobrr.com/snippets/freeleech.mdx"

NAMED_GROUP_RE = re.compile(r"\(\?P<([A-Za-z_][A-Za-z0-9_]*)>")


def captured_vars(parse: Dict[str, Any]) -> Set[str]:
    """Return the fields the line patterns of a channel capture."""
    captured: Set[str] = set()

    for line in parse.get("lines") or []:
        if not isinstance(line, dict) or line.get("ignore"):
            continue

        # an explicit vars list keeps the legacy positional mapping and takes
        # precedence over named groups, same as IndexerIRCParseLine.ParseLine
        explicit = line.get("vars")
        if explicit:
            captured.update(v for v in explicit if isinstance(v, str))
            continue

        captured.update(NAMED_GROUP_RE.findall(line.get("pattern") or ""))

    return captured


def mapped_vars(parse: Dict[str, Any]) -> Tuple[Set[str], List[float]]:
    """Return the fields a channel maps captured values to, and the download volume factors among them."""
    mapped: Set[str] = set()
    factors: List[float] = []

    for values in (parse.get("mappings") or {}).values():
        for targets in (values or {}).values():
            if not isinstance(targets, dict):
                continue

            mapped.update(targets)

            factor = targets.get("downloadVolumeFactor")
            try:
                factors.append(float(factor))
            except (TypeError, ValueError):
                continue

    return mapped, factors


def parse_freeleech(parse: Dict[str, Any]) -> Tuple[bool, bool]:
    """Return whether a channel announces freeleech and freeleech percent."""
    captured = captured_vars(parse)
    mapped, factors = mapped_vars(parse)
    produced = captured | mapped

    # a factor below 1 is inverted into a freeleech percent by Release.MapVars, so
    # a mapping that reaches 0 (fully free) fills in both fields, not just freeleech
    discounted = any(0 <= f < 1 for f in factors)

    freeleech = "freeleech" in produced or discounted
    freeleech_percent = "freeleechPercent" in produced or discounted

    # captured directly instead of mapped, so any value can be announced
    if "downloadVolumeFactor" in captured:
        freeleech = freeleech_percent = True

    return freeleech, freeleech_percent


def parse_definition(file_path: str) -> Dict[str, Any]:
    """Read an indexer definition and reduce it to the fields the docs tables need."""
    with open(file_path, "r", encoding="utf-8") as f:
        definition = yaml.safe_load(f)

    if not isinstance(definition, dict):
        raise ValueError("definition is not a mapping")

    supports = [s.lower() for s in definition.get("supports") or [] if isinstance(s, str)]

    freeleech = False
    freeleech_percent = False

    for channel in ((definition.get("irc") or {}).get("channels") or []):
        parse = (channel or {}).get("parse")
        if not parse:
            continue

        channel_freeleech, channel_freeleech_percent = parse_freeleech(parse)
        freeleech = freeleech or channel_freeleech
        freeleech_percent = freeleech_percent or channel_freeleech_percent

    return {
        "name": str(definition.get("name") or "").strip(),
        "description": str(definition.get("description") or "").strip(),
        "supports": supports,
        "freeleech": freeleech,
        "freeleechPercent": freeleech_percent,
    }


def get_feature_checkmark(value: bool) -> str:
    return "✓" if value else "✗"


def escape_cell(value: str) -> str:
    return value.replace("|", "\\|")


def generate_indexers_markdown(indexers: list) -> str:
    """Generate markdown for indexers table."""
    markdown = "<details>\n\n"
    markdown += "<summary>Click to view supported indexers</summary>\n\n"
    markdown += "| Indexer | Description | IRC | RSS |\n"
    markdown += "|---------|-------------|-----|-----|\n"

    for indexer in indexers:
        name = escape_cell(indexer.get("name", ""))
        description = escape_cell(indexer.get("description", ""))
        irc_support = get_feature_checkmark("irc" in indexer.get("supports", []))
        rss_support = get_feature_checkmark("rss" in indexer.get("supports", []))

        markdown += f"| {name} | {description} | {irc_support} | {rss_support} |\n"

    markdown += "\n</details>"
    return markdown


def generate_freeleech_markdown(indexers: list) -> str:
    """Generate markdown for freeleech table."""
    markdown = "| Indexer | Freeleech | Freeleech Percent |\n"
    markdown += "|---------|-----------|------------------|\n"

    for indexer in indexers:
        if not (indexer.get("freeleech", False) or indexer.get("freeleechPercent", False)):
            continue

        name = escape_cell(indexer.get("name", ""))
        freeleech = get_feature_checkmark(indexer.get("freeleech", False))
        freeleech_percent = get_feature_checkmark(indexer.get("freeleechPercent", False))

        markdown += f"| {name} | {freeleech} | {freeleech_percent} |\n"

    return markdown


def main():
    """Generate markdown documents"""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--definitions", default=DEFINITIONS_DIR, help="directory with indexer definitions")
    parser.add_argument("--indexers-output", default=INDEXERS_OUTPUT, help="path of the generated indexers table")
    parser.add_argument("--freeleech-output", default=FREELEECH_OUTPUT, help="path of the generated freeleech table")
    args = parser.parse_args()

    indexers = []
    failed = []

    for filename in sorted(os.listdir(args.definitions)):
        if not filename.endswith(".yaml"):
            continue

        file_path = os.path.join(args.definitions, filename)
        try:
            indexers.append(parse_definition(file_path))
        except (OSError, ValueError, yaml.YAMLError) as e:
            failed.append(f"{filename}: {e}")

    if failed:
        print("Could not parse indexer definitions:", file=sys.stderr)
        for failure in failed:
            print(f"  {failure}", file=sys.stderr)
        return 1

    # Sort indexers by name, but put generic ones last
    def sort_key(indexer):
        name = indexer.get("name", "").lower()
        return (name.startswith("generic"), name)

    indexers.sort(key=sort_key)

    for output_file, content in [
        (args.indexers_output, generate_indexers_markdown(indexers)),
        (args.freeleech_output, generate_freeleech_markdown(indexers)),
    ]:
        os.makedirs(os.path.dirname(output_file), exist_ok=True)
        with open(output_file, "w", encoding="utf-8") as f:
            f.write(content)

    return 0


if __name__ == "__main__":
    sys.exit(main())
