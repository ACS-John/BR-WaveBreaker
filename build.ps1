# Builds WaveBreaker.exe as a GUI-subsystem binary (no console window) with
# debug info stripped, so it stays small and never flashes a console.
$ErrorActionPreference = "Stop"
Push-Location $PSScriptRoot
try {
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"

    # Regenerate the embedded-icon resource (rsrc_windows_amd64.syso) from the app's own
    # ACS-v5.ico so the .exe's file icon and the tray icon both use it (resource ID 1,
    # loaded via LoadIconW(hInstance, 1) in tray.go). go build auto-links any *.syso file
    # found in this directory -- no separate linking step needed.
    go run github.com/akavel/rsrc@latest -ico "..\..\Icon\ACS-v5.ico" -o rsrc_windows_amd64.syso -arch amd64

    go build -buildvcs=false -ldflags "-H=windowsgui -s -w" -o WaveBreaker.exe .
    Write-Host "Built $((Get-Item WaveBreaker.exe).FullName) ($((Get-Item WaveBreaker.exe).Length) bytes)"
} finally {
    Pop-Location
}
