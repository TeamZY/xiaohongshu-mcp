//go:build stdio && windows
// +build stdio,windows

package main

import (
	"os"
	"syscall"
)

// redirectStdoutToStderr 在 Windows 上把物理 fd 1（STDOUT）重定向到 NUL（黑洞），
// 避免 go-rod 的 launcher 把进度日志写入底层 fd 1 而污染 MCP 的 stdout。
//
// Windows 下 syscall.Dup/Dup2 不可用，因此使用 Win32 API (GetStdHandle/SetStdHandle)
// 实现等效的 fd 级重定向：
//  1. 保存原始 stdout 句柄供 MCP StdioTransport 使用；
//  2. 打开 NUL（Windows 的 /dev/null），把 STD_OUTPUT_HANDLE 指向它；
//  3. os.Stdout 变量保持指向原始 stdout（MCP 协议使用）。
//
// 这样 go-rod 及其子进程写入物理 fd 1 的日志全部被丢弃，
// 而 MCP 的 JSON-RPC 响应仍通过 os.Stdout 正确输出。
func redirectStdoutToStderr() (*os.File, error) {
	// 1. 保存原始 os.Stdout 文件对象（指向真实的 stdout 句柄）
	originalStdout := os.Stdout

	// 2. 打开 NUL（Windows 的 /dev/null）
	nul, err := os.OpenFile("NUL", os.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}

	// 3. 使用 Win32 API 把 STD_OUTPUT_HANDLE (fd 1) 重定向到 NUL
	//    这样 go-rod/launcher 等通过 GetStdHandle(STD_OUTPUT_HANDLE) 写入
	//    的日志全部被丢弃，不会污染 MCP 的 stdout。
	const STD_OUTPUT_HANDLE = ^uintptr(11 - 1) // -11 as unsigned
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setStdHandle := kernel32.NewProc("SetStdHandle")
	r1, _, err := setStdHandle.Call(STD_OUTPUT_HANDLE, nul.Fd())
	if r1 == 0 {
		nul.Close()
		return nil, err
	}

	// 不需要关闭 nul，因为进程退出时会自动关闭
	// 返回 nil 表示没有需要额外关闭的文件（os.Stdout 保持不变）
	_ = originalStdout // os.Stdout 保持不变，指向原始 stdout
	return nil, nil
}
