//go:build stdio && unix
// +build stdio,unix

package main

import (
	"os"
	"syscall"
)

// redirectStdoutToStderr 在 fd（文件描述符）层面重定向 stdout。
//
// 为什么需要这样做（而不是简单的 os.Stdout = os.Stderr）：
//   - go-rod/rod 的 launcher 在启动/下载浏览器时会把进度日志直接写入底层 fd 1，
//     而不是 Go 的 os.Stdout 变量。简单的 os.Stdout = os.Stderr 无法拦截。
//   - MCP stdio 协议要求 os.Stdout 必须只承载 JSON-RPC 消息。
//
// 本函数：
//  1. 在 fd 层面把真实 stdout（fd 1）复制一份，包装为 *os.File 供 MCP 协议使用；
//  2. 把 fd 1（物理 stdout）dup 到 fd 2（stderr），于是 rod 写入 fd 1 的日志全部进入 stderr；
//  3. 让 os.Stdout 变量指向真正的 stdout 副本（供 StdioTransport 使用）。
func redirectStdoutToStderr() (*os.File, error) {
	// 1. 复制真实 stdout，供 MCP 协议后续使用
	fd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}
	realStdout := os.NewFile(uintptr(fd), "stdout")
	// 2. 把 fd 1 重定向到 fd 2（stderr），让 rod 的 fd 级日志进 stderr
	if err := syscall.Dup2(int(os.Stderr.Fd()), 1); err != nil {
		return nil, err
	}
	// 3. 让 Go 的 os.Stdout 变量指向真正的 stdout 副本（供 StdioTransport 使用）
	os.Stdout = realStdout
	return realStdout, nil
}
