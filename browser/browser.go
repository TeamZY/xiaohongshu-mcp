package browser

import (
	"encoding/json"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/go-rod/rod/lib/defaults"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
)

type browserConfig struct {
	binPath string
}

type Option func(*browserConfig)

func WithBinPath(binPath string) Option {
	return func(c *browserConfig) {
		c.binPath = binPath
	}
}

// maskProxyCredentials masks username and password in proxy URL for safe logging.
func maskProxyCredentials(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil || u.User == nil {
		return proxyURL
	}
	if _, hasPassword := u.User.Password(); hasPassword {
		u.User = url.UserPassword("***", "***")
	} else {
		u.User = url.User("***")
	}
	return u.String()
}

// parseCookiesString 将纯字符串格式的 cookies 转换为 JSON 数组格式
// 输入格式: "abRequestId=xxx; ets=xxx; a1=xxx"
// 输出格式: [{"name":"abRequestId","value":"xxx","domain":".xiaohongshu.com"},...]
func parseCookiesString(cookiesStr string) string {
	// 如果已经是 JSON 数组格式，直接返回
	if strings.HasPrefix(strings.TrimSpace(cookiesStr), "[") {
		return cookiesStr
	}

	// 解析纯字符串格式
	parts := strings.Split(cookiesStr, ";")
	var cookieList []map[string]string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		name := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		cookie := map[string]string{
			"name":   name,
			"value":  value,
			"domain": ".xiaohongshu.com",
		}
		cookieList = append(cookieList, cookie)
	}

	if len(cookieList) == 0 {
		return ""
	}

	jsonData, err := json.Marshal(cookieList)
	if err != nil {
		logrus.Warnf("failed to marshal cookies to JSON: %v", err)
		return ""
	}

	return string(jsonData)
}

// detectSystemChromium 在当前系统中探测已安装的 Chromium/Chrome 可执行文件。
// 不依赖任何外部配置（命令行参数或环境变量），解决线上未指定 -bin 时
// go-rod 自动下载浏览器、把进度日志写入 fd 1 污染 MCP stdout 的问题。
// 找不到时返回空字符串（由上层决定后续行为，但不会触发下载日志）。
func detectSystemChromium() string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files\Chromium\Application\chrome.exe`,
		}
	} else {
		candidates = []string{
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chrome",
			"/snap/bin/chromium",
			"/opt/google/chrome/chrome",
		}
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			// 进一步确认有可执行权限（Unix 下）
			if runtime.GOOS != "windows" {
				if info.Mode()&0111 == 0 {
					continue
				}
			}
			return c
		}
	}
	return ""
}

func NewBrowser(headless bool, options ...Option) *headless_browser.Browser {
	cfg := &browserConfig{}
	for _, opt := range options {
		opt(cfg)
	}

	// 兜底：确保 rod 使用系统已安装的 Chromium，避免运行时自动下载。
	// 自动下载会把进度日志写入 fd 1，污染 MCP stdio 的 stdout（aios 会因此
	// 报 JSONRPCMessage 解析失败）。
	//
	// 优先级：显式 binPath > ROD_BROWSER_BIN 环境变量 > 自动探测系统浏览器。
	// 注意：go-rod v0.116.2 使用 defaults.Bin 作为全局默认浏览器路径
	//（launcher.DefaultBrowserBinPath 是该变量在更新版本才引入）。
	//
	// 关闭自动下载行为：go-rod v0.116.2 中只要 defaults.Bin 非空就不会触发
	// 浏览器下载。下面的优先级链保证 Bin 一旦被设置即用系统浏览器，
	// 绝不触发下载流程，从而避免任何 fd 1 日志污染。

	if cfg.binPath != "" {
		defaults.Bin = cfg.binPath
	} else if envBin := os.Getenv("ROD_BROWSER_BIN"); envBin != "" {
		defaults.Bin = envBin
	} else if detected := detectSystemChromium(); detected != "" {
		// 未显式指定时自动探测系统已安装的 Chromium/Chrome，
		// 无需任何外部配置即可避免自动下载。
		defaults.Bin = detected
		logrus.Infof("auto-detected system Chromium: %s", detected)
	}

	opts := []headless_browser.Option{
		headless_browser.WithHeadless(headless),
	}
	if cfg.binPath != "" {
		opts = append(opts, headless_browser.WithChromeBinPath(cfg.binPath))
	}

	// Read proxy from environment variable
	if proxy := os.Getenv("XHS_PROXY"); proxy != "" {
		opts = append(opts, headless_browser.WithProxy(proxy))
		logrus.Infof("Using proxy: %s", maskProxyCredentials(proxy))
	}

	// 加载 cookies（优先从环境变量 COOKIES 获取）
	if data, err := cookies.GetCookies(); err == nil {
		// 将纯字符串格式转换为 JSON 数组格式
		cookiesJSON := parseCookiesString(string(data))
		if cookiesJSON != "" {
			opts = append(opts, headless_browser.WithCookies(cookiesJSON))
			logrus.Info("loaded cookies successfully")
		} else {
			logrus.Warn("failed to parse cookies to JSON format")
		}
	} else {
		logrus.Warnf("failed to load cookies: %v", err)
	}

	return headless_browser.New(opts...)
}
