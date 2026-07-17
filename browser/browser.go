package browser

import (
	"encoding/json"
	"net/url"
	"os"
	"strings"

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

func NewBrowser(headless bool, options ...Option) *headless_browser.Browser {
	cfg := &browserConfig{}
	for _, opt := range options {
		opt(cfg)
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
