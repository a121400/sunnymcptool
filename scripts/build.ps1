# SunnyNet build script
# Usage: .\build.ps1 | .\build.ps1 -Target gui | .\build.ps1 -Target mcp | .\build.ps1 -Clean

param(
    [ValidateSet("all", "gui", "mcp")]
    [string]$Target = "all",
    [switch]$Clean,
    [switch]$NoCompress
)

$ErrorActionPreference = "Stop"
$OutputDir = "build\bin"
$Version = "1.0.4.0"

function Write-Step($msg) {
    Write-Host "[Build] $msg" -ForegroundColor Cyan
}

function Write-Ok($msg) {
    Write-Host "[OK] $msg" -ForegroundColor Green
}

function Write-Fail($msg) {
    Write-Host "[FAIL] $msg" -ForegroundColor Red
}

if ($Clean) {
    Write-Step "Cleaning..."
    if (Test-Path $OutputDir) {
        Remove-Item "$OutputDir\SunnyNet.exe" -Force -ErrorAction SilentlyContinue
        Remove-Item "$OutputDir\sunnynet-mcp.exe" -Force -ErrorAction SilentlyContinue
    }
    Write-Ok "Clean done."
}

if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

Write-Step "Checking env..."
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Fail "Go not found. Install Go first."
    exit 1
}
Write-Host "  Go: $goVersion"

function Compress-WithUPX($filePath) {
    if ($NoCompress) { return }
    $upx = Get-Command upx -ErrorAction SilentlyContinue
    if (-not $upx) { return }
    $before = (Get-Item $filePath).Length / 1MB
    Write-Step "Compressing $(Split-Path $filePath -Leaf) with UPX..."
    & $upx.Source --best --lzma $filePath 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        $after = (Get-Item $filePath).Length / 1MB
        $ratio = [math]::Round(($before - $after) / $before * 100, 1)
        Write-Ok ("Compressed: " + [string]([math]::Round($before, 2)) + " MB -> " + [string]([math]::Round($after, 2)) + " MB (-${ratio}%)")
    }
}

function Build-MCP {
    Write-Step "Building MCP (sunnynet-mcp.exe, static)..."
    $ldflags = "-s -w -X main.Version=$Version -extldflags=-static"
    $env:CGO_ENABLED = "1"
    $env:CGO_LDFLAGS = "-static"
    go build -ldflags $ldflags -trimpath -o "$OutputDir\sunnynet-mcp.exe" ./mcp_standalone/
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "MCP build failed."
        exit 1
    }
    $size = (Get-Item "$OutputDir\sunnynet-mcp.exe").Length / 1MB
    Write-Ok ("sunnynet-mcp.exe " + [string]([math]::Round($size, 2)) + " MB")
    Compress-WithUPX "$OutputDir\sunnynet-mcp.exe"
}

function Build-GUI {
    $wailsVer = wails version 2>&1 | Select-String -Pattern "v\d" | ForEach-Object { $_.ToString().Trim() }
    if (-not $wailsVer) {
        Write-Fail "Wails CLI not found. Run: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    }
    Write-Host "  Wails: $wailsVer"
    Write-Step "Building SunnyNet GUI (static)..."
    $env:CGO_ENABLED = "1"
    $env:CGO_LDFLAGS = "-static"
    wails build -clean -trimpath
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "GUI build failed."
        exit 1
    }
    $size = (Get-Item "$OutputDir\SunnyNet.exe").Length / 1MB
    Write-Ok ("SunnyNet.exe " + [string]([math]::Round($size, 2)) + " MB")
    Compress-WithUPX "$OutputDir\SunnyNet.exe"
}

switch ($Target) {
    "mcp" { Build-MCP }
    "gui" { Build-GUI }
    "all" {
        Build-GUI
        Build-MCP
    }
}

Write-Host ""
Write-Step "Output: $OutputDir"
Get-ChildItem $OutputDir | ForEach-Object {
    $sizeMB = $_.Length / 1MB
    $sz = [string]([math]::Round($sizeMB, 2))
    Write-Host ("  " + $_.Name.PadRight(30) + " " + $sz + " MB")
}
Write-Host ""
Write-Ok "Build done."
