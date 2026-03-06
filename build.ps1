# SunnyNet 构建脚本
# 用法:
#   .\build.ps1              - 构建全部 (主程序 + MCP独立服务)
#   .\build.ps1 -Target gui  - 仅构建 Wails GUI 主程序
#   .\build.ps1 -Target mcp  - 仅构建 MCP 独立服务
#   .\build.ps1 -Clean       - 清理后重新构建

param(
    [ValidateSet("all", "gui", "mcp")]
    [string]$Target = "all",
    [switch]$Clean
)

$ErrorActionPreference = "Stop"
$OutputDir = "build\bin"
$Version = "1.0.2.8"

function Write-Step($msg) {
    Write-Host "[构建] $msg" -ForegroundColor Cyan
}

function Write-Ok($msg) {
    Write-Host "[成功] $msg" -ForegroundColor Green
}

function Write-Fail($msg) {
    Write-Host "[失败] $msg" -ForegroundColor Red
}

# 清理
if ($Clean) {
    Write-Step "清理构建产物..."
    if (Test-Path $OutputDir) {
        # 只删除 exe 文件，保留 dll
        Remove-Item "$OutputDir\SunnyNet.exe" -Force -ErrorAction SilentlyContinue
        Remove-Item "$OutputDir\sunnynet-mcp.exe" -Force -ErrorAction SilentlyContinue
    }
    Write-Ok "清理完成"
}

# 确保输出目录存在
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# 检查工具链
Write-Step "检查构建环境..."
$goVersion = go version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Fail "未找到 Go 编译器，请先安装 Go"
    exit 1
}
Write-Host "  Go: $goVersion"


# 构建 MCP 独立服务
function Build-MCP {
    Write-Step "编译 MCP 独立服务 (sunnynet-mcp.exe)..."

    $ldflags = "-s -w -X main.Version=$Version"
    go build -ldflags $ldflags -trimpath -o "$OutputDir\sunnynet-mcp.exe" ./mcp_standalone/

    if ($LASTEXITCODE -ne 0) {
        Write-Fail "MCP 独立服务编译失败"
        exit 1
    }

    $size = (Get-Item "$OutputDir\sunnynet-mcp.exe").Length / 1MB
    Write-Ok "sunnynet-mcp.exe ({0:N2} MB)" -f $size
}

# 构建 Wails GUI 主程序
function Build-GUI {
    # 检查 wails
    $wailsVer = wails version 2>&1 | Select-String -Pattern "v\d" | ForEach-Object { $_.ToString().Trim() }
    if (-not $wailsVer) {
        Write-Fail "未找到 Wails CLI，请先安装: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    }
    Write-Host "  Wails: $wailsVer"

    Write-Step "编译 SunnyNet 主程序 (Wails GUI)..."
    wails build -clean -trimpath

    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Wails 主程序编译失败"
        exit 1
    }

    $size = (Get-Item "$OutputDir\SunnyNet.exe").Length / 1MB
    Write-Ok "SunnyNet.exe ({0:N2} MB)" -f $size
}

# 执行构建
# 注意: wails build -clean 会清空 bin 目录，所以 MCP 必须在 GUI 之后编译
switch ($Target) {
    "mcp" {
        Build-MCP
    }
    "gui" {
        Build-GUI
    }
    "all" {
        Build-GUI
        Build-MCP
    }
}

# 列出产物
Write-Host ""
Write-Step "构建产物 ($OutputDir):"
Get-ChildItem $OutputDir | ForEach-Object {
    $sizeMB = $_.Length / 1MB
    Write-Host ("  {0,-30} {1,8:N2} MB" -f $_.Name, $sizeMB)
}

Write-Host ""
Write-Ok "构建完成"
