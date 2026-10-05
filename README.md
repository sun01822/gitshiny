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

On Linux / macOS, if your shell says `gitshiny: command not found`, the install directory is not on your `PATH`. Add it once.

Installed with curl:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Installed with Go:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Using bash? Replace `~/.zshrc` with `~/.bashrc`. If `go env GOPATH` prints something other than `~/go`, use that path plus `/bin` instead. `command -v gitshiny` prints the binary's path once it is found.

To install somewhere else, set `GITSHINY_INSTALL_DIR`:

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | GITSHINY_INSTALL_DIR="$HOME/bin" sh
```

**Windows** (PowerShell, no admin rights needed, installs to `%LOCALAPPDATA%\Programs\gitshiny`, verifies SHA-256):

```powershell
[Net.ServicePointManager]::SecurityProtocol = 'Tls12'; irm https://raw.githubusercontent.com/sun01822/gitshiny/main/install.ps1 | iex
```

Paste it as one line into **PowerShell**. From the old Command Prompt (`cmd.exe`) use this instead:

```bat
powershell -NoProfile -Command "[Net.ServicePointManager]::SecurityProtocol = 'Tls12'; irm https://raw.githubusercontent.com/sun01822/gitshiny/main/install.ps1 | iex"
```

Then go to a Git repository and run `gitshiny`.

- `gitshiny` is not recognized? Close the terminal app completely (Windows Terminal, VS Code, ...) and open it again; a new tab is not always enough to pick up the changed `Path`.
- `gitshiny.exe` is not an installer. Double-clicking it only shows these instructions; it is meant to be run from a terminal.
- If the install fails, the last line says which step failed and why. Please include it when you open an issue.
- GitShiny needs [Git for Windows](https://git-scm.com/download/win) on your `Path`.
- To install somewhere else, run `$env:GITSHINY_INSTALL_DIR = "C:\Tools\gitshiny"` before the install command.
- Prefer to do it by hand? Download `gitshiny_windows_amd64.zip` from the [Releases page](https://github.com/sun01822/gitshiny/releases) and unzip `gitshiny.exe` into any folder that is on your `Path`.
- `go install` works on Windows too; the binary lands in `%USERPROFILE%\go\bin`.
- Only 64-bit Intel/AMD (amd64) is built for Windows.

## Update

Already installed? Check what you have, then install again over it — the installer always fetches the latest release and replaces the binary in place.

```bash
gitshiny version
```

Installed with curl — re-run the installer:

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | sh
```

Installed with Go:

```bash
go install github.com/sun01822/gitshiny@latest
```

To get a specific version (this is also how you downgrade):

```bash
curl -fsSL https://raw.githubusercontent.com/sun01822/gitshiny/main/install.sh | GITSHINY_VERSION=v0.2.0 sh
```

Windows: run the install command again; it replaces `gitshiny.exe` with the latest release. For a specific version, set it first:

```powershell
$env:GITSHINY_VERSION = "v0.5.0"
```

## Uninstall

GitShiny is a single binary. Delete it and it is gone.

Installed with curl:

```bash
rm ~/.local/bin/gitshiny
```

Installed with Go:

```bash
rm "$(go env GOPATH)/bin/gitshiny"
```

Check it is gone — this prints nothing once it is removed:

```bash
command -v gitshiny
```

- Not sure where it is? Run `command -v gitshiny` first; it prints the path to delete.
- Installed with `GITSHINY_INSTALL_DIR`? Delete `gitshiny` from that directory.
- There is nothing else to clean up: GitShiny creates no config, cache or data files, and the installer never edits your shell config. If you added the `export PATH=...` line to `~/.zshrc` only for GitShiny you can remove it by hand, but other tools often use `~/.local/bin` and `~/go/bin`, so leave it if unsure.

**Windows** (PowerShell) — delete the install folder, then take it off your user `Path`:

```powershell
$dir = "$env:LOCALAPPDATA\Programs\gitshiny"
Remove-Item -Recurse -Force $dir
$path = [Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ -and $_ -ne $dir }
[Environment]::SetEnvironmentVariable('Path', ($path -join ';'), 'User')
```

The last two lines remove only the GitShiny folder from `Path`; every other entry is kept. Close the terminal app and open it again, then `gitshiny version` should say `gitshiny` is not recognized.

- "The process cannot access the file" error? A `gitshiny` is still running; close it and run the commands again.
- Installed with `GITSHINY_INSTALL_DIR`? Put that folder in the first line instead.
- Installed with `go install`? Run `Remove-Item "$env:USERPROFILE\go\bin\gitshiny.exe"`; there is no `Path` entry to remove.
- Prefer clicking? Delete the folder in File Explorer, then open Settings, search for "Edit environment variables for your account", edit `Path` and delete the `gitshiny` entry.
- There is nothing else to clean up on Windows either: no config, cache, registry keys or Start-menu entries.

## Interactive TUI

Run `gitshiny` with no arguments in any Git repository.

```text
╭────────────────────────────────────────────────────╮
│                                                    │
│   ✦ GitShiny                      gitshiny · main  │
│                                                    │
│  TIME RANGE                                        │
│  ▸ 1  Today        Fri, Oct 2                      │
│    2  Yesterday    Thu, Oct 1                      │
│    3  Custom       pick any start and end          │
│                                                    │
│  AUTHOR                                            │
│  sun01822                                          │
│                                                    │
│  ↑/↓ move · enter select · 1-3 jump · q quit       │
│                                                    │
╰────────────────────────────────────────────────────╯
```

```text
╭────────────────────────────────────────────────────╮
│                                                    │
│   ✦ GitShiny                      gitshiny · main  │
│                                                    │
│  Today · sun01822                                  │
│  2026-10-02 00:00 → 2026-10-02 23:59               │
│                                                    │
│  ╭─────────────╮ ╭─────────────╮ ╭─────────────╮   │
│  │ +2,152      │ │ -141        │ │ +2,011      │   │
│  │ added       │ │ removed     │ │ net growth  │   │
│  ╰─────────────╯ ╰─────────────╯ ╰─────────────╯   │
│  ████████████████████████████████████████████▒▒▒   │
│                                                    │
│  Commits  7              Files changed  25         │
│                                                    │
│  updated 18:20:39                                  │
│                                                    │
│  r refresh · esc menu · q quit                     │
│                                                    │
╰────────────────────────────────────────────────────╯
```

In a real terminal it is in colour: the selected row and title are filled with the accent colour, added lines are green, removed lines are red, and the bar shows added against removed. Colours adapt to light and dark terminals and are switched off by `NO_COLOR`.

| Key | Action |
|---|---|
| `↑` `↓` or `j` `k` | move (wraps around) |
| `1` `2` `3` | jump straight to Today, Yesterday, Custom |
| `enter` | select, next field, calculate |
| `tab` | switch between Start and End |
| `r` | refresh, or retry after an error |
| `esc` | back to the menu |
| `q` | quit with a short thank-you animation (any key skips it) |
| `ctrl+c` | quit immediately |

- **Custom** takes the same time formats as `--since` / `--until` below; the fields support cursor movement and paste.
- While GitShiny reads the history it shows a spinner and a sliding bar. Git usually answers in milliseconds, so the TUI holds that screen for about 0.7 s to keep it visible; `gitshiny stats` is never delayed.
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
install.sh       curl installer (Linux / macOS)
install.ps1      PowerShell installer (Windows)
```

The statistics engine never touches the UI: the CLI and the Bubble Tea TUI both call it through the same `compute` function.

## Develop

```bash
go vet ./...
go test ./...
go build -o gitshiny .
./gitshiny
./gitshiny stats --today
```

`./gitshiny` opens the TUI; `./gitshiny stats --today` runs the CLI.

## Publish a release

Once, to replace the placeholder owner name:

```bash
scripts/set-owner.sh YOUR_GITHUB_NAME
```

Then tag and push; GitHub Actions builds and publishes:

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
```

The workflow tests, builds Linux/macOS (amd64, arm64) and Windows (amd64) archives, writes `checksums.txt`, and creates the GitHub Release the installer downloads from. `go install` works as soon as the repo is public and tagged.

## License

MIT
