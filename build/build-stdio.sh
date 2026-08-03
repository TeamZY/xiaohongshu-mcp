#!/usr/bin/env bash
# ============================================================
# xhs-mcp-stdio 跨平台编译脚本 (bash)
# 用于生成本地两个二进制：Linux 版 + Windows 版
# 修复内容：rod 浏览器下载进度日志污染 MCP stdout 的问题
# ============================================================
set -euo pipefail
cd "$(dirname "$0")/.."

build_one() {
    local goos="$1" goarch="$2" output="$3"
    echo ">>> Building $output (GOOS=$goos GOARCH=$goarch) ..."
    GOOS="$goos" GOARCH="$goarch" go build -tags stdio -trimpath -o "$output" .
    echo "<<< OK: $output"
}

# Linux 版（线上 Docker 容器用，带 fd 级重定向修复）
build_one "linux"   "amd64" "xhs-mcp-stdio"
# Windows 版（本地 dev 用）
build_one "windows" "amd64" "xhs-mcp-stdio.exe"

echo ""
echo "All builds done."
ls -lh xhs-mcp-stdio xhs-mcp-stdio.exe
