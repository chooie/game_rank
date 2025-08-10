#!/usr/bin/env pwsh
# scripts/run.ps1
$ErrorActionPreference = 'Stop'

# ──────────────────────────────────────────────────────────────────────────────
# Paths
# ──────────────────────────────────────────────────────────────────────────────
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$ROOT_DIR   = (Resolve-Path "$SCRIPT_DIR\..").Path
$SRC_DIR    = Join-Path $ROOT_DIR 'src'
$DIST_CSS   = Join-Path $ROOT_DIR 'public\dist\css'
$SERVER     = Join-Path $SRC_DIR 'server.js'
$ENV_FILE   = Join-Path $SCRIPT_DIR '.env'

# ──────────────────────────────────────────────────────────────────────────────
# .env loader (simple key=value; preserves spaces, strips wrapping quotes)
# ──────────────────────────────────────────────────────────────────────────────
if (Test-Path $ENV_FILE) {
  Get-Content $ENV_FILE | ForEach-Object {
    if ($_ -match '^\s*(#|$)') { return }
    $kv = $_ -split '=', 2
    if ($kv.Count -eq 2) {
      $k = $kv[0].Trim()
      $v = $kv[1]
      if ($v -match '^\s*"(.*)"\s*$') { $v = $Matches[1] }
      elseif ($v -match "^\s*'(.*)'\s*$") { $v = $Matches[1] }
      else { $v = $v.Trim() }
      [System.Environment]::SetEnvironmentVariable($k, $v)
    }
  }
}

if (-not $env:NODE_ENV) { $env:NODE_ENV = 'development' }

# ──────────────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────────────
function To-PosixPath([string]$p) { $p -replace '\\','/' }
function Get-SassInOutPair() {
  "$(To-PosixPath("$SRC_DIR/templates")):$(To-PosixPath($DIST_CSS))"
}

function Start-Proc([string]$file, [string[]]$argList) {
  $psi = [System.Diagnostics.ProcessStartInfo]::new()
  $psi.FileName = $file
  $psi.WorkingDirectory = $ROOT_DIR
  $psi.UseShellExecute = $false
  foreach ($a in $argList) { $null = $psi.ArgumentList.Add($a) }
  [System.Diagnostics.Process]::Start($psi)
}

function Exec([string]$file, [string[]]$argList) {
  $psi = [System.Diagnostics.ProcessStartInfo]::new()
  $psi.FileName = $file
  $psi.WorkingDirectory = $ROOT_DIR
  $psi.UseShellExecute = $false
  foreach ($a in $argList) { $null = $psi.ArgumentList.Add($a) }
  $p = [System.Diagnostics.Process]::Start($psi)
  $p.WaitForExit()
  if ($p.ExitCode -ne 0) {
    $joined = ($argList -join ' ')
    throw "Process exited with code $($p.ExitCode): $file $joined"
  }
}

# Node + JS entry points (avoid .cmd shims)
$NODE       = 'node'
$SASS_JS    = Join-Path $ROOT_DIR 'node_modules\sass\sass.js'
$NODEMON_JS = Join-Path $ROOT_DIR 'node_modules\nodemon\bin\nodemon.js'

# ──────────────────────────────────────────────────────────────────────────────
# Tasks
# ──────────────────────────────────────────────────────────────────────────────
function SassDev {
  if (-not (Test-Path $SASS_JS)) {
    Write-Host "ℹ️  sass not found; skipping SCSS watch. Run: npm i -D sass"
    return $null
  }
  New-Item -ItemType Directory -Force -Path $DIST_CSS | Out-Null
  $pair = Get-SassInOutPair
  $args = @($SASS_JS, '--watch', $pair, '--style=expanded', '--embed-source-map', '--quiet')
  Start-Proc $NODE $args
}

function SassProd {
  if (-not (Test-Path $SASS_JS)) {
    Write-Host "ℹ️  sass not found; skipping SCSS build. Run: npm i -D sass"
    return
  }
  New-Item -ItemType Directory -Force -Path $DIST_CSS | Out-Null
  $pair = Get-SassInOutPair
  $args = @($SASS_JS, $pair, '--style=compressed', '--no-source-map', '--quiet')
  Exec $NODE $args
}

function DevServer {
  if (Test-Path $NODEMON_JS) {
    # Use -x/--exec with separate args so quoting is reliable
    $args = @(
      $NODEMON_JS,
      '-w', $SRC_DIR,
      '-e', 'js,json,hbs,handlebars,scss',
      '-x', 'node', '--', $SERVER
    )
    Exec $NODE $args
  } else {
    Write-Host "⚠️  nodemon not found; starting plain node (no auto-reload)."
    Exec $NODE @($SERVER)
  }
}

function ProdServer {
  Exec $NODE @($SERVER)
}

# ──────────────────────────────────────────────────────────────────────────────
# Main
# ──────────────────────────────────────────────────────────────────────────────
if ($env:NODE_ENV -eq 'development') {
  $sassProc = SassDev
  try {
    DevServer
  } finally {
    if ($sassProc) {
      try { $sassProc.Kill() | Out-Null } catch {}
    }
  }
} else {
  SassProd
  ProdServer
}
