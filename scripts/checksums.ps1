$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root 'dist'
if (-not (Test-Path -LiteralPath $dist)) { throw 'dist/ is missing' }
$lines = @()
Get-ChildItem -LiteralPath $dist -File | Where-Object { $_.Name -notlike 'SHA256SUMS*' } | ForEach-Object {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $_.FullName).Hash.ToLower()
    $lines += "$hash  $($_.Name)"
}
Set-Content -LiteralPath (Join-Path $dist 'SHA256SUMS.txt') -Value ($lines -join "`n") -Encoding ascii
Write-Output ($lines -join "`n")
