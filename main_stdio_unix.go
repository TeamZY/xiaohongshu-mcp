//go:build stdio && unix
// +build stdio,unix

package main

import (
	"os"
	"syscall"
)

// redirectStdoutToStderr 在 fd（文件描述符）层面隔离 MCP 协议的 stdout。
//
// 为什么需要这样做（而不是简单的 os.Stdout = os.Stderr）：
//   - go-rod/rod 的 launcher 在启动/下载浏览器时会把进度日志直接写入底层 fd 1，
//     而不是 Go 的 os.Stdout 变量。而且 launcher 内部会 exec 一个中间进程、
//     重新 dup fd，简单的 os.Stdout = os.Stderr 无法拦截，dup 到 stderr 也会
//     被子进程继承关系绕过，最终导致 [launcher.Browser]...Progress 日志混进
//     MCP 的 stdout，使 aios 解析 JSON-RPC 失败。
//   - MCP stdio 协议要求 os.Stdout 必须只承载 JSON-RPC 消息。
//
// 本函数的稳健策略：
//  1. 先把真实 stdout（fd 1）复制一份，包装为 *os.File 供 MCP 协议（StdioTransport）使用；
//  2. 把物理 fd 1 彻底重定向到 /dev/null（黑洞），rod 及其子进程写入 fd 1 的
//     所有日志一律丢弃，绝不可能再污染 MCP 的 stdout；
//  3. 让 Go 的 os.Stdout 变量指向步骤 1 保留的干净 stdout 副本。
//
// 注意：日志（logrus/slog）仍在 main 中通过 MultiWriter 写到 stderr + 文件，
// 不受影响；只有 rod 的 fd 级原生输出被丢弃。
func redirectStdoutToStderr() (*os.File, error) {
	// 1. 复制真实 stdout，供 MCP 协议后续使用（指向物理 stdout）
	fd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return nil, err
	}
	realStdout := os.NewFile(uintptr(fd), "stdout")

	// 2. 打开 /dev/null，把物理 fd 1 重定向到它（黑洞）。
	//    这样 rod/浏览器子进程写入 fd 1 的 [launcher.Browser] 进度日志全部被丢弃，
	//    不会再进入 MCP 的 stdout。
	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	if err != nil {
		// /dev/null 不可用（极少见）时退化为重定向到 stderr，避免崩溃
		if dupErr := syscall.Dup2(int(os.Stderr.Fd()), 1); dupErr != nil {
			return nil, dupErr
		}
	} else {
		if dupErr := syscall.Dup2(int(devNull.Fd()), 1); dupErr != nil {
			devNull.Close()
			return nil, dupErr
		}
		devNull.Close()
	}

	// 3. 让 Go 的 os.Stdout 变量指向真正的 stdout 副本（供 StdioTransport 使用）
	os.Stdout = realStdout
	return realStdout, nil
}
