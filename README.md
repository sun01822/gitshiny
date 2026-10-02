# GitShiny

Git contribution statistics for your terminal — added lines, removed lines, net growth, commits and files changed for **today**, **yesterday** or **any time range**. Use the interactive TUI, or the `stats` command for scripts and CI. No `git log | awk` gymnastics. Single static binary, no runtime dependencies.

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

To install somewhere else, set `GITSHINY_INSTALL_DIR`:

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | GITSHINY_INSTALL_DIR="$HOME/bin" sh
```

Windows: download the `.zip` from the [Releases page](https://github.com/sun01822/gitshiny/releases).

## Update

Already installed? Check what you have, then install again over it — the installer always fetches the latest release and replaces the binary in place.

```bash
gitshiny version

# installed with curl: re-run the installer
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | sh

# installed with Go
go install github.com/sun01822/gitshiny@latest
```

To get a specific version (this is also how you downgrade):

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | GITSHINY_VERSION=v0.2.0 sh
```

Windows: download the newer `.zip` and replace `gitshiny.exe`.

## Uninstall

GitShiny is a single binary. Delete it and it is gone:

```bash
rm ~/.local/bin/gitshiny                  # installed with curl
rm "$(go env GOPATH)/bin/gitshiny"        # installed with Go
command -v gitshiny                       # prints nothing once it is removed
```

- Not sure where it is? Run `command -v gitshiny` first; it prints the path to delete.
- Installed with `GITSHINY_INSTALL_DIR`? Delete `gitshiny` from that directory.
- There is nothing else to clean up: GitShiny creates no config, cache or data files, and the installer never edits your shell config. If you added the `export PATH=...` line to `~/.zshrc` only for GitShiny you can remove it by hand, but other tools often use `~/.local/bin`, so leave it if unsure.
- Windows: delete `gitshiny.exe`.

## Interactive TUI

Run `gitshiny` with no arguments in any Git repository.

```text
╭──────────────────────────────────────╮
│               GitShiny               │
│ ──────────────────────────────────── │
│                                      │
│ Time Range                           │
│ > Today                              │
│   Yesterday                          │
│   Custom                             │
│                                      │
│ Author                               │
│ sun01822                             │
│                                      │
│ ↑/↓ move  enter select  q quit       │
╰──────────────────────────────────────╯
```

```text
╭──────────────────────────────────────╮
│               GitShiny               │
│ ──────────────────────────────────── │
│                                      │
│ Repository   gitshiny                │
│ Branch       main                    │
│ Author       sun01822                │
│ Period       Today                   │
│                                      │
│ Added                 1,653          │
│ Removed                  28          │
│ Net Growth           +1,625          │
│ Commits                   4          │
│ Files Changed            24          │
│                                      │
│ r refresh  esc menu  q quit          │
╰──────────────────────────────────────╯
```

| Key | Action |
|---|---|
| `↑` `↓` or `j` `k` | move |
| `enter` | select, next field, calculate |
| `tab` | switch between Start and End |
| `r` | refresh, or retry after an error |
| `esc` | back to the menu |
| `q` / `ctrl+c` | quit |

- **Custom** takes the same time formats as `--since` / `--until` below.
- Quit from the statistics screen and the result is printed as plain text, so it stays in your scrollback.
- The TUI starts only when both input and output are a terminal. Piped or redirected runs print the usage text instead, so scripts never hang on it.

## CLI

For scripts, CI and one-off checks.

```bash
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
go vet ./...
go test ./...
go build -o gitshiny .
./gitshiny                  # TUI
./gitshiny stats --today    # CLI
```

## Publish a release

```bash
scripts/set-owner.sh YOUR_GITHUB_NAME   # once, replaces the placeholder
git tag vX.Y.Z && git push origin vX.Y.Z    # GitHub Actions builds + publishes
```

The workflow tests, builds Linux/macOS (amd64, arm64) and Windows (amd64) archives, writes `checksums.txt`, and creates the GitHub Release the installer downloads from. `go install` works as soon as the repo is public and tagged.

## License

MIT
