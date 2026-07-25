#!/usr/bin/env bash
#
# Update the project's Go toolchain and Docker base images to the latest versions.
#
# Auto-detects the latest stable Go, the newest Node LTS, and the newest Alpine from
# go.dev / nodejs.org / Docker Hub, then rewrites:
#   - go.mod                  `go` directive   -> latest stable Go minor   (e.g. go 1.26.0)
#   - every Dockerfile/*.Dockerfile in the repo (root or nested, e.g. distrib/docker/):
#       golang:<tag>  image   -> golang:<minor>-alpine<X>   (e.g. 1.26-alpine3.24)
#       node:<tag>    image   -> node:<LTS>-alpine<X>       (e.g. 24.18.0-alpine3.24)
#       alpine:<tag>  image   -> alpine:<X.Y>               (e.g. 3.24)
#
# Image references are matched as tokens, so they are updated regardless of a
# `--platform=...` prefix, an `AS <stage>` suffix, or appearing inside a LABEL.
# Non-matching base images (e.g. mcr.microsoft.com/windows/nanoserver) are left as-is,
# and a Dockerfile that lacks a given image is simply skipped for that image.
#
# Usage:
#   scripts/update-toolchain.sh            # detect and apply the edits in place
#   scripts/update-toolchain.sh --check    # detect and print only; make no changes
#
# Requires: bash, curl, jq, GNU sed, grep, coreutils sort (-V). Runs from any CWD in the repo.
#
# When run inside GitHub Actions (GITHUB_OUTPUT is set) it also writes the resolved
# values and a `changed` flag as step outputs for the workflow to consume.

set -euo pipefail

CHECK_ONLY=false
case "${1:-}" in
  --check | -n | --dry-run) CHECK_ONLY=true ;;
  -h | --help)
    sed -n '2,/^set -euo/p' "${BASH_SOURCE[0]}" | sed '$d; s/^# \{0,1\}//'
    exit 0
    ;;
  "") ;;
  *)
    echo "unknown argument: $1" >&2
    echo "usage: $0 [--check]" >&2
    exit 2
    ;;
esac

in_ci() { [ -n "${GITHUB_ACTIONS:-}" ]; }

die() {
  if in_ci; then echo "::error::$1"; else echo "error: $1" >&2; fi
  exit 1
}

for bin in curl jq sed grep sort awk find; do
  command -v "$bin" >/dev/null 2>&1 || die "required command not found: $bin"
done

# Operate from the repository root so relative paths resolve regardless of CWD.
repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[ -n "$repo_root" ] || repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

[ -f go.mod ] || die "go.mod not found in $repo_root"

# Every Dockerfile / *.Dockerfile in the repo (e.g. ./Dockerfile and
# ./distrib/docker/ci.Dockerfile), skipping vendored/generated trees.
dockerfiles=()
while IFS= read -r -d '' f; do
  dockerfiles+=("${f#./}")
done < <(
  find . -type f \( -name Dockerfile -o -name '*.Dockerfile' \) \
    -not -path './.git/*' \
    -not -path '*/node_modules/*' \
    -not -path '*/vendor/*' \
    -not -path '*/testdata/*' \
    -print0 | sort -z
)
[ "${#dockerfiles[@]}" -gt 0 ] || die "no Dockerfile(s) found under $repo_root"

# ---------------------------------------------------------------------------
# Detect latest versions
# ---------------------------------------------------------------------------

# Latest stable Go (e.g. "go1.26.4" -> "1.26.4", minor "1.26").
go_full="$(curl -fsSL 'https://go.dev/dl/?mode=json' \
  | jq -r 'map(select(.stable == true)) | .[0].version' || true)"
go_version="${go_full#go}"
[[ "$go_version" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] \
  || die "could not determine latest stable Go release (got '$go_full')"
go_minor="$(echo "$go_version" | cut -d. -f1,2)"
# Pin the go.mod directive to the minor (matches the minor-pinned builder image
# and avoids churn on every patch release).
go_mod_version="${go_minor}.0"

# Newest alpine variant of golang:<minor>-alpine<X.Y>.
golang_alpine="$(curl -fsSL "https://hub.docker.com/v2/repositories/library/golang/tags/?page_size=100&name=${go_minor}-alpine" \
  | jq -r '.results[].name' \
  | grep -E "^${go_minor//./\\.}-alpine[0-9]+\.[0-9]+$" \
  | sed -E 's/^.*-alpine//' \
  | sort -V | tail -1 || true)"
[ -n "$golang_alpine" ] || die "no golang:${go_minor}-alpine* image tag found on Docker Hub"
golang_tag="${go_minor}-alpine${golang_alpine}"

# Newest Node LTS full version (e.g. "24.18.0").
node_version="$(curl -fsSL 'https://nodejs.org/dist/index.json' \
  | jq -r 'map(select(.lts != false)) | .[].version' \
  | sed 's/^v//' \
  | sort -V | tail -1 || true)"
[[ "$node_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] \
  || die "could not determine latest Node LTS (got '$node_version')"

# Newest alpine variant of node:<version>-alpine<X.Y>.
node_alpine="$(curl -fsSL "https://hub.docker.com/v2/repositories/library/node/tags/?page_size=100&name=${node_version}-alpine" \
  | jq -r '.results[].name' \
  | grep -E "^${node_version//./\\.}-alpine[0-9]+\.[0-9]+$" \
  | sed -E 's/^.*-alpine//' \
  | sort -V | tail -1 || true)"
[ -n "$node_alpine" ] || die "no node:${node_version}-alpine* image tag found on Docker Hub"
node_tag="${node_version}-alpine${node_alpine}"

# Alpine minor for runner/base images: newest variant shipped by the builder images
# above, so the runtime base stays consistent with (and never ahead of) the build
# toolchain. Both inputs are bounded single-page queries, avoiding a paginated
# library/alpine lookup.
alpine_version="$(printf '%s\n%s\n' "$golang_alpine" "$node_alpine" | sort -V | tail -1)"
[[ "$alpine_version" =~ ^[0-9]+\.[0-9]+$ ]] \
  || die "could not determine Alpine version (got '$alpine_version')"

# ---------------------------------------------------------------------------
# Current values (representative, for reporting) — tolerant of absent images
# ---------------------------------------------------------------------------
old_go="$(grep -oE '^go [0-9][^[:space:]]*' go.mod | awk '{print $2}' || true)"
old_golang="$(grep -hoE 'golang:[^[:space:]"]+' "${dockerfiles[@]}" 2>/dev/null | head -1 || true)"
old_node="$(grep -hoE 'node:[^[:space:]"]+' "${dockerfiles[@]}" 2>/dev/null | head -1 || true)"
old_alpine="$(grep -hoE 'alpine:[^[:space:]"]+' "${dockerfiles[@]}" 2>/dev/null | head -1 || true)"

# ---------------------------------------------------------------------------
# Compute & apply edits (in-place unless --check)
# ---------------------------------------------------------------------------
any_changed=false
report=()

apply() { # file  transformed-content
  local f="$1" new="$2"
  if [ "$new" != "$(cat "$f")" ]; then
    any_changed=true
    report+=("$f: updated")
    "$CHECK_ONLY" || printf '%s\n' "$new" >"$f"
  else
    report+=("$f: no change")
  fi
}

# go.mod `go` directive
apply go.mod "$(sed -E "s|^go [0-9]+\.[0-9]+(\.[0-9]+)?$|go ${go_mod_version}|" go.mod)"

# Image tokens across every Dockerfile
for f in "${dockerfiles[@]}"; do
  apply "$f" "$(sed -E \
    -e "s|golang:[^[:space:]\"]+|golang:${golang_tag}|g" \
    -e "s|node:[^[:space:]\"]+|node:${node_tag}|g" \
    -e "s|alpine:[^[:space:]\"]+|alpine:${alpine_version}|g" \
    "$f")"
done

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------
row() { # label  old  new
  local note=""
  [ "$2" = "$3" ] && note="  (no change)"
  printf '  %-16s %s -> %s%s\n' "$1" "$2" "$3" "$note"
}

echo "Detected latest versions:"
row "go (go.mod)" "$old_go" "$go_mod_version"
row "golang image" "${old_golang:-n/a}" "golang:${golang_tag}"
row "node image" "${old_node:-n/a}" "node:${node_tag}"
row "alpine image" "${old_alpine:-n/a}" "alpine:${alpine_version}"
echo
echo "Files (${#dockerfiles[@]} Dockerfile(s) + go.mod):"
for line in "${report[@]}"; do echo "  $line"; done
echo

# ---------------------------------------------------------------------------
# Emit step outputs when running under GitHub Actions
# ---------------------------------------------------------------------------
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "changed=$any_changed"
    echo "go_mod_version=$go_mod_version"
    echo "golang_tag=$golang_tag"
    echo "node_tag=$node_tag"
    echo "alpine_version=$alpine_version"
    echo "old_go=$old_go"
    echo "old_golang=$old_golang"
    echo "old_node=$old_node"
    echo "old_alpine=$old_alpine"
  } >>"$GITHUB_OUTPUT"
fi

if "$CHECK_ONLY"; then
  echo "--check: no files were modified."
elif "$any_changed"; then
  echo "Applied updates."
else
  echo "Already up to date — nothing to change."
fi
