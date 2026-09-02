#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob

base_ref="${1:?base git revision is required}"
definitions_dir="internal/indexer/definitions"
deprecated_dir="${definitions_dir}/deprecated"

identifier_from_yaml() {
  sed -n 's/^identifier:[[:space:]]*//p' | head -n 1 | tr -d "'\""
}

require_active_or_tombstone() {
  local identifier="$1"
  local source_file="$2"

  if [[ -z "${identifier}" ]]; then
    echo "Could not read the identifier from base definition ${source_file}" >&2
    missing=1
    return
  fi

  local active_file
  for active_file in "${definitions_dir}"/*.yaml; do
    if [[ "$(identifier_from_yaml < "${active_file}")" == "${identifier}" ]]; then
      return
    fi
  done

  local tombstone="${deprecated_dir}/${identifier}.yaml"
  if [[ ! -f "${tombstone}" ]]; then
    echo "Removed indexer identifier ${identifier} requires ${tombstone}" >&2
    missing=1
  fi
}

missing=0
while IFS= read -r base_file; do
  [[ -z "${base_file}" ]] && continue
  identifier="$(git show "${base_ref}:${base_file}" | identifier_from_yaml)"
  require_active_or_tombstone "${identifier}" "${base_file}"
done < <(git ls-tree -r --name-only "${base_ref}" "${definitions_dir}" | awk -F/ 'NF == 4 && $NF ~ /\.yaml$/')

while IFS= read -r base_file; do
  [[ -z "${base_file}" ]] && continue
  identifier="$(git show "${base_ref}:${base_file}" | identifier_from_yaml)"
  require_active_or_tombstone "${identifier}" "${base_file}"
done < <(git ls-tree -r --name-only "${base_ref}" "${deprecated_dir}" | awk -F/ 'NF == 5 && $NF ~ /\.yaml$/')

exit "${missing}"
