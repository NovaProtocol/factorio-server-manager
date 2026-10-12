#!/bin/sh
# Copy the companion mod source into the manager's embedded copy.
#
# The companion-mod repository is the source of truth; src/companion/mod is a
# vendored copy that the manager embeds with //go:embed. Run this after changing
# the mod, then commit both repositories.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
src="$here/../../companion-mod"
dst="$here/../src/companion/mod"

if [ ! -f "$src/info.json" ]; then
  echo "companion mod source not found at $src" >&2
  exit 1
fi

mkdir -p "$dst"
cp "$src/info.json" "$src/control.lua" "$dst/"
echo "synced companion mod into $dst"
