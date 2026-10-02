#!/usr/bin/env bash
# GitShiny Phase 1 - Bash MVP (kept for reference; the Go CLI replaces it)
set -e

AUTHOR=$(git config user.name 2>/dev/null || true)
if [ -z "$AUTHOR" ]; then
    echo "Error: git config user.name is not configured."
    echo
    echo "Run:"
    echo '  git config --global user.name "Your Name"'
    exit 1
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: This directory is not a Git repository."
    exit 1
fi

# GNU date uses -d, BSD/macOS date uses -v.
yesterday() {
    date -d 'yesterday' "+$1" 2>/dev/null || date -v-1d "+$1"
}

echo
echo "========================================"
echo "              GitShiny"
echo "========================================"
echo
echo "Author: $AUTHOR"
echo
echo "Select time range:"
echo
echo "  1) Today"
echo "  2) Yesterday"
echo "  3) Custom"
echo

read -rp "Enter your choice [1-3]: " CHOICE

case "$CHOICE" in
    1)
        SINCE="$(date '+%Y-%m-%d 00:00:00')"
        UNTIL="$(date '+%Y-%m-%d 23:59:59')"
        PERIOD="Today"
        ;;
    2)
        SINCE="$(yesterday '%Y-%m-%d 00:00:00')"
        UNTIL="$(yesterday '%Y-%m-%d 23:59:59')"
        PERIOD="Yesterday"
        ;;
    3)
        echo
        read -rp "Start [YYYY-MM-DD HH:MM:SS]: " SINCE
        read -rp "End   [YYYY-MM-DD HH:MM:SS]: " UNTIL
        PERIOD="Custom"
        ;;
    *)
        echo
        echo "Error: Invalid option."
        exit 1
        ;;
esac

echo
echo "Period : $PERIOD"
echo "Since  : $SINCE"
echo "Until  : $UNTIL"
echo

STATS=$(
    git log \
        --fixed-strings \
        --author="$AUTHOR" \
        --since="$SINCE" \
        --until="$UNTIL" \
        --pretty=tformat: \
        --numstat |
    awk '
    {
        if ($1 ~ /^[0-9]+$/ && $2 ~ /^[0-9]+$/) {
            added += $1
            removed += $2
        }
    }
    END {
        printf "%d %d %d\n", added, removed, added - removed
    }'
)

read -r ADDED REMOVED NET <<< "$STATS"

echo "========================================"
echo "             Git Statistics"
echo "========================================"
echo
printf "Added lines    : %s\n" "$ADDED"
printf "Removed lines  : %s\n" "$REMOVED"
printf "Net growth     : %s\n" "$NET"
echo
echo "========================================"
