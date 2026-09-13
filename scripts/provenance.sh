#!/usr/bin/env bash
# provenance.sh — coverage report for externally-sourced facts (docs/comment-contract.md).
#
# Not a CI gate. It answers two questions:
#   1. Which provenance-bearing definition sites have no `Source:` tag?
#   2. Which tags are still `inferred` / `stale?` (i.e. need re-checking against upstream)?
#
# A new harness may land with `stale?` — it may not land without a Source line.

set -uo pipefail
cd "$(dirname "$0")/.."

have_rg=0
command -v rg >/dev/null 2>&1 && have_rg=1

grep_or_rg() { # pattern file
  if [ "$have_rg" = 1 ]; then rg -n "$1" "$2" 2>/dev/null; else grep -nE "$1" "$2" 2>/dev/null; fi
}

exists() { [ "$have_rg" = 1 ] && rg -q "$1" "$2" 2>/dev/null || grep -qE "$1" "$2" 2>/dev/null; }

# Sites where a wrong value means writing an invalid file into someone's tool.
SITES="
internal/skills/paths.go
internal/mcp/target.go
internal/mcp/translate.go
internal/modelsdev/modelsdev.go
internal/modelsdev/defaults.go
internal/modelfile/normalize.go
internal/openrouter/openrouter.go
internal/sync/r2.go
internal/catalog/catalog.go
internal/catalog/project.go
internal/core/merge.go
internal/secrets/providers.go
internal/provider/builtin/providers.yaml
internal/adapter/adapter.go
"
for f in internal/adapters/*/*.go internal/provider/*.go; do
  case "$f" in *_test.go) continue ;; esac
  SITES="$SITES $f"
done

missing=0
ok=0
printf '== Source: coverage on provenance-bearing sites ==\n'
for f in $SITES; do
  [ -f "$f" ] || continue
  if exists 'Source:[[:space:]]+(upstream|vendor-doc|vendor-api|local-measure|internal)' "$f"; then
    ok=$((ok + 1))
  else
    printf '  MISSING  %s\n' "$f"
    missing=$((missing + 1))
  fi
done
printf '  %d tagged, %d missing\n\n' "$ok" "$missing"

printf '== Tags needing re-check (inferred / stale?) ==\n'
if [ "$have_rg" = 1 ]; then
  rg -n 'Source:.*(inferred|stale\?)' internal cmd 2>/dev/null | sed 's/^/  /'
else
  grep -rnE 'Source:.*(inferred|stale\?)' internal cmd 2>/dev/null | sed 's/^/  /'
fi

printf '\n== Magic deviations and hard constraints ==\n'
for tag in 'Magic deviation:' 'Never:' 'Invariant:' 'TODO(FUTURE)'; do
  if [ "$have_rg" = 1 ]; then
    n=$(rg -c "$tag" -g '*.go' -g '*.yaml' internal cmd 2>/dev/null | awk -F: '{s+=$2} END {print s+0}')
  else
    n=$(grep -rE "$tag" --include='*.go' --include='*.yaml' internal cmd 2>/dev/null | wc -l | tr -d ' ')
  fi
  printf '  %-16s %s\n' "$tag" "$n"
done

[ "$missing" -gt 0 ] && exit 1
exit 0
