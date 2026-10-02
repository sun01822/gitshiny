# GitShiny

Git contribution statistics for your terminal — added lines, removed lines, net growth, commits and files changed for **today**, **yesterday** or **any time range**. No `git log | awk` gymnastics. Single static binary, no runtime dependencies.

## Install

**curl** (Linux / macOS, installs to `~/.local/bin`, verifies SHA-256):

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | sh
```

**Go** (1.23+):

```bash
go install github.com/sun01822/gitshiny@latest
```

Make sure `~/.local/bin` (curl) or `$(go env GOPATH)/bin` (Go) is on your `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"   # add to ~/.zshrc
```

Windows: download the `.zip` from the Releases page.

## Usage

Run it in any Git repository.

```bash
gitshiny                      # interactive TUI
gitshiny stats --today
gitshiny stats --yesterday
gitshiny stats --since "2026-10-01 09:00:00" --until "2026-10-01 18:00:00"
gitshiny stats --today --format json
gitshiny stats --yesterday --author Alice --author Bob --format csv
gitshiny stats --all-authors --all-branches --since 2026-10-01
```

| Flag | Meaning |
|---|---|
| `--today` / `--yesterday` | preset ranges (default: today) |
| `--since` / `--until` | custom range: `YYYY-MM-DD HH:MM:SS`, `YYYY-MM-DD HH:MM`, `YYYY-MM-DD`, or RFC3339 |
| `--author NAME` | repeatable or comma separated; default is `git config user.name` |
| `--all-authors` | include everybody |
| `--branch NAME` / `--all-branches` | default is the current branch |
| `--format text\|json\|csv` | default `text` (or set `GITSHINY_FORMAT`) |
| `-C DIR` | run as if started in `DIR` |

TUI keys: `↑`/`↓` (or `j`/`k`) move, `enter` select, `tab` next field, `r` refresh, `esc` back, `q` quit. The TUI only starts on a terminal; piped or redirected runs print the usage text.

JSON output (handy for CI):

```json
{
  "repository": "gitshiny",
  "author": "Sun",
  "branch": "main",
  "period": "Today",
  "since": "2026-10-02 00:00:00",
  "until": "2026-10-02 23:59:59",
  "added": 842,
  "removed": 213,
  "net_growth": 629,
  "commits": 12,
  "files_changed": 37
}
```

Notes: merge commits are ignored; binary files count toward *files changed* but not lines; author matching is a substring match (git's `--author` semantics); times use your local time zone unless you pass an RFC3339 offset.

## Layout

```
main.go          entry point
cmd/             CLI parsing and output (text/json/csv)
config/          environment configuration
domain/          core models
services/        statistics engine (UI independent)
repositories/    git access
tui/             interactive Bubble Tea UI
utils/           time parsing, number formatting
scripts/         bash MVP, release build, set-owner
install.sh       curl installer
```

The statistics engine never touches the UI: the CLI and the Bubble Tea TUI both call it through the same `compute` function.

## Develop

```bash
go test ./...
go build -o gitshiny . && ./gitshiny stats --today
```

## Publish a release

```bash
scripts/set-owner.sh YOUR_GITHUB_NAME   # once, replaces the placeholder
git tag v0.1.0 && git push origin v0.1.0    # GitHub Actions builds + publishes
```

The workflow tests, builds Linux/macOS (amd64, arm64) and Windows (amd64) archives, writes `checksums.txt`, and creates the GitHub Release the installer downloads from. `go install` works as soon as the repo is public and tagged.

## License

MIT
