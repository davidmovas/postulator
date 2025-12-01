# Build script for Postulator with version injection
# Usage: .\scripts\build.ps1 [-Version "1.0.0"] [-Release]

param(
    [string]$Version = "",
    [switch]$Release = $false
)

$ErrorActionPreference = "Stop"

# Get version from package.json if not provided
if (-not $Version) {
    $packageJson = Get-Content -Path "frontend\package.json" | ConvertFrom-Json
    $Version = $packageJson.version
}

# Get git commit hash
$Commit = git rev-parse --short HEAD 2>$null
if (-not $Commit) {
    $Commit = "unknown"
}

# Get build date
$BuildDate = Get-Date -Format "yyyy-MM-dd"

Write-Host "Building Postulator" -ForegroundColor Cyan
Write-Host "  Version:    $Version" -ForegroundColor Green
Write-Host "  Commit:     $Commit" -ForegroundColor Green
Write-Host "  Build Date: $BuildDate" -ForegroundColor Green
Write-Host ""

# Construct ldflags
$ldflags = "-X github.com/davidmovas/postulator/internal/version.Version=$Version -X github.com/davidmovas/postulator/internal/version.Commit=$Commit -X github.com/davidmovas/postulator/internal/version.BuildDate=$BuildDate"

if ($Release) {
    # Production build - hide console window
    $ldflags = "$ldflags -H windowsgui -s -w"
    Write-Host "Building RELEASE version (no console)" -ForegroundColor Yellow
    wails build -ldflags "$ldflags" -platform windows/amd64
} else {
    # Development build - keep console for debugging
    Write-Host "Building DEBUG version (with console)" -ForegroundColor Yellow
    wails build -ldflags "$ldflags"
}

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "Build successful!" -ForegroundColor Green
    Write-Host "Output: build\bin\Postulator.exe" -ForegroundColor Cyan

    # Show file info
    $exePath = "build\bin\Postulator.exe"
    if (Test-Path $exePath) {
        $fileInfo = Get-Item $exePath
        Write-Host "Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Gray
    }
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
