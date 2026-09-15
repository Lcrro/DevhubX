$ErrorActionPreference = 'Stop'
$dest = Join-Path $env:LOCALAPPDATA 'Programs\DevHub'
if (Test-Path -LiteralPath $dest) {
    Remove-Item -LiteralPath $dest -Recurse -Force
    Write-Output "Removed $dest"
} else {
    Write-Output 'DevHub program files were not found.'
}
$data = Join-Path $env:APPDATA 'DevHub'
Write-Output "Project files were not touched. Data directory left in place: $data"
