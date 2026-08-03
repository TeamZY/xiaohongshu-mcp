//go:build stdio && windows
// +build stdio,windows

package main

import (
	"os"
)

// redirectStdoutToStderr 在 Windows 上把 os.Stdout 临时指向 os.Stderr，
// 避免 rod 浏览器启动日志干扰 MCP stdio 的 stdout 协议消息。
//
// Windows 下 syscall.Dup/Dup2 不可用，且 Windows 本地运行场景对 rod 的
// fd 级日志污染不敏感，因此采用简单的 os.Stdout 变量重定向即可。
// 返回 nil 表示无需额外关闭的文件。
func redirectStdoutToStderr() (*os.File, error) {
	os.Stdout = os.Stderr
	return nil, nil
}
