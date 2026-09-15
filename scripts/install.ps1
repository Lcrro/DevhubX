$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$exe = Join-Path $root 'dist\devhub.exe'
if (-not (Test-Path -LiteralPath $exe)) {
    throw 'Build dist/devhub.exe first (scripts/build.ps1).'
}
$dest = Join-Path $env:LOCALAPPDATA 'Programs\DevHub'
New-Item -ItemType Directory -Force $dest | Out-Null
Copy-Item -LiteralPath $exe -Destination (Join-Path $dest 'devhub.exe') -Force
$wsh = New-Object -ComObject WScript.Shell
$shortcut = $wsh.CreateShortcut((Join-Path $dest 'DevHub.lnk'))
$shortcut.TargetPath = Join-Path $dest 'devhub.exe'
$shortcut.WorkingDirectory = $dest
$shortcut.Save()
Write-Output "Installed $dest\devhub.exe"
Write-Output "Data stays in $env:APPDATA\DevHub and is not modified by setup."
