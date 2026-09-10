#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/magic"
FIX="$ROOT/testdata/demo-home"
LIB="$ROOT/testdata/demo-lib"
rm -rf "$FIX" "$LIB"
mkdir -p "$FIX/.pi/agent/skills/demo" "$FIX/.pi/agent/sessions"
echo '{"theme":"dark"}' > "$FIX/.pi/agent/settings.json"
echo 'x = 1' > "$FIX/.pi/agent/my-extra.toml"
echo '{}' > "$FIX/.pi/agent/sessions/s.json"
echo '# skill' > "$FIX/.pi/agent/skills/demo/SKILL.md"

export MAGIC_HOME="$LIB"
"$BIN" init
echo "=== dry-run (expect unknown my-extra.toml) ==="
"$BIN" capture --dry-run --home "$FIX" --only pi
echo "=== approve ==="
"$BIN" capture approve pi my-extra.toml
echo "=== capture ==="
"$BIN" capture --home "$FIX" --only pi
echo "=== library tree ==="
find "$LIB/packs" -type f | sort
