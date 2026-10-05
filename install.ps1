# GitShiny installer for Windows (Windows PowerShell 5.1 and PowerShell 7)
#   [Net.ServicePointManager]::SecurityProtocol = 'Tls12'; irm https://raw.githubusercontent.com/sun01822/gitshiny/main/install.ps1 | iex
#
# Environment overrides:
#   GITSHINY_VERSION      install a specific tag (default: latest release)
#   GITSHINY_INSTALL_DIR  install location (default: %LOCALAPPDATA%\Programs\gitshiny)
#
# This runs through iex in the user's own session, so: no param(), no exit
# (it would close their window), and everything stays inside one script block
# so the preference variables below do not leak out.
& {
  $ErrorActionPreference = 'Stop'
  $ProgressPreference = 'SilentlyContinue'

  $repo = 'sun01822/gitshiny'
  $file = 'gitshiny_windows_amd64.zip'
  $dir = if ($env:GITSHINY_INSTALL_DIR) { $env:GITSHINY_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\gitshiny" }
  $version = $env:GITSHINY_VERSION
  if ($version -and $version -notlike 'v*') { $version = "v$version" }
  $base = if ($version) { "https://github.com/$repo/releases/download/$version" } else { "https://github.com/$repo/releases/latest/download" }

  $tmp = Join-Path ([IO.Path]::GetTempPath()) "gitshiny-install-$PID"
  $step = 'starting'
  Write-Host 'Installing GitShiny...'
  try {
    if (-not [Environment]::Is64BitOperatingSystem) { throw 'GitShiny is only built for 64-bit Windows' }
    # Windows PowerShell 5.1 can default to TLS 1.0, which GitHub refuses.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    New-Item -ItemType Directory -Force -Path $tmp | Out-Null

    $step = "downloading $base/$file"
    Invoke-WebRequest -UseBasicParsing -Uri "$base/$file" -OutFile "$tmp\$file"
    Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile "$tmp\checksums.txt"
    Write-Host '  ok  Downloaded'

    $step = 'verifying the checksum'
    $line = Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match "\s\*?$([regex]::Escape($file))\s*$" } | Select-Object -First 1
    if (-not $line) { throw "no checksum listed for $file" }
    $expected = ($line.Trim() -split '\s+')[0]
    $actual = (Get-FileHash "$tmp\$file" -Algorithm SHA256).Hash
    if ($actual -ne $expected) { throw "checksum mismatch (expected $expected, got $actual)" }
    Write-Host '  ok  Checksum verified'

    $step = "unpacking to $dir (close any running gitshiny first)"
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Expand-Archive -Path "$tmp\$file" -DestinationPath $dir -Force
    Write-Host "  ok  Installed $dir\gitshiny.exe"

    $step = 'adding the folder to your Path'
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (($userPath -split ';') -notcontains $dir) {
      $newPath = if ($userPath) { $userPath.TrimEnd(';') + ';' + $dir } else { $dir }
      [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
      Write-Host '  ok  Added to your user Path'
    }
    if (($env:Path -split ';') -notcontains $dir) { $env:Path = $env:Path.TrimEnd(';') + ';' + $dir }

    $step = 'running gitshiny'
    $installed = & "$dir\gitshiny.exe" version
    Write-Host ''
    Write-Host "$installed installed successfully."
    Write-Host ''
    Write-Host 'Open a terminal inside a Git repository and run:'
    Write-Host ''
    Write-Host '    gitshiny'
    Write-Host ''
    Write-Host 'Other terminals that are already open must be closed and reopened to find it.'
  } catch {
    throw "GitShiny install failed while ${step}: $($_.Exception.Message)"
  } finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
  }
}
