param([int]$Port = 0)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$server = Join-Path $projectRoot 'roadmap\server.mjs'
if ($Port -gt 0) {
  & node $server --port $Port
} else {
  & node $server
}
