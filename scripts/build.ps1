$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$goTool = Get-Command go -ErrorAction SilentlyContinue
$goExecutable = if ($goTool) { $goTool.Source } else { Join-Path $projectRoot '.tools\go\bin\go.exe' }
if (-not (Test-Path -LiteralPath $goExecutable)) { throw 'Go 1.26+ is required. Install it from https://go.dev/dl/' }
Push-Location (Join-Path $projectRoot 'web')
try {
    npm ci
    if ($LASTEXITCODE -ne 0) { throw 'npm ci failed' }
    npm run build
    if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed' }
} finally { Pop-Location }
Push-Location $projectRoot
try {
    New-Item -ItemType Directory -Force dist | Out-Null
    & $goExecutable build -trimpath -o dist/devhub.exe ./cmd/devhub
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }
} finally { Pop-Location }
