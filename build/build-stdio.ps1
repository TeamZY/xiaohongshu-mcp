# ============================================================
# xhs-mcp-stdio 跨平台编译脚本 (PowerShell)
# 用于生成本地两个二进制：Linux 版 + Windows 版
# 修复内容：rod 浏览器下载进度日志污染 MCP stdout 的问题
#   - fd 级 stdout->stderr 重定向 (Linux, main_stdio_unix.go)
#   - defaults.Bin 全局浏览器路径兜底，避免 rod 自动下载
# ============================================================

$ProjectDir = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $ProjectDir

function Build-One($goos, $goarch, $output) {
    Write-Host ">>> Building $output (GOOS=$goos GOARCH=$goarch) ..." -ForegroundColor Cyan
    $env:GOOS = $goos
    $env:GOARCH = $goarch
    & go build -tags stdio -trimpath -o $output .
    $exit = $LASTEXITCODE
    if ($exit -ne 0) {
        Write-Host "<<< FAILED: $output (exit=$exit)" -ForegroundColor Red
        exit $exit
    }
    Write-Host "<<< OK: $output" -ForegroundColor Green
}

# Linux 版（线上 Docker 容器用，带 fd 级重定向修复）
Build-One "linux"   "amd64" "xhs-mcp-stdio"
# Windows 版（本地 dev 用）
Build-One "windows" "amd64" "xhs-mcp-stdio.exe"

# 清理交叉编译环境变量，避免影响后续命令
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "All builds done." -ForegroundColor Green
Get-ChildItem (Join-Path $ProjectDir "xhs-mcp-stdio"), (Join-Path $ProjectDir "xhs-mcp-stdio.exe") |
    Select-Object Name, Length, LastWriteTime | Format-Table -AutoSize
