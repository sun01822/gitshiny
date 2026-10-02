#!/bin/sh
# Replace the USERNAME placeholder with your GitHub username.
# Usage: scripts/set-owner.sh your-github-username
set -eu
OWNER="${1:?usage: set-owner.sh <github-username>}"
cd "$(dirname "$0")/.."
grep -rl --exclude-dir=.git --exclude=set-owner.sh 'USERNAME' . | while read -r f; do
  sed "s/USERNAME/$OWNER/g" "$f" > "$f.tmp" && cat "$f.tmp" > "$f" && rm "$f.tmp"
done
echo "Updated placeholder to: $OWNER"
echo "Next: go build ./... && go test ./..."
