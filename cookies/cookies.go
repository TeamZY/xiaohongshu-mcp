package cookies

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Cookier interface {
	LoadCookies() ([]byte, error)
	SaveCookies(data []byte) error
	DeleteCookies() error
}

type localCookie struct {
	path string
}

func NewLoadCookie(path string) Cookier {
	if path == "" {
		panic("path is required")
	}

	return &localCookie{
		path: path,
	}
}

// LoadCookies 从文件中加载 cookies。
func (c *localCookie) LoadCookies() ([]byte, error) {

	data, err := os.ReadFile(c.path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read cookies from tmp file")
	}

	return data, nil
}

// SaveCookies 保存 cookies 到文件中。
func (c *localCookie) SaveCookies(data []byte) error {
	return os.WriteFile(c.path, data, 0644)
}

// DeleteCookies 删除 cookies 文件。
func (c *localCookie) DeleteCookies() error {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		// 文件不存在，返回 nil（认为已经删除）
		return nil
	}
	return os.Remove(c.path)
}

// GetCookies 获取 cookies 内容。
// 优先级：
// 1. 环境变量 COOKIES（直接包含 cookies 内容）
// 2. 环境变量 COOKIES_PATH 指定的文件
// 3. /tmp/cookies.json 文件（向后兼容）
// 4. 当前目录下的 cookies.json 文件
func GetCookies() ([]byte, error) {
	// 优先级 1：从环境变量 COOKIES 直接获取
	if cookies := os.Getenv("COOKIES"); cookies != "" {
		logrus.Infof("GetCookies: 从环境变量 COOKIES 获取，长度=%d", len(cookies))
		// 打印前100个字符用于调试
		if len(cookies) > 100 {
			logrus.Debugf("GetCookies: COOKIES 内容前100字符: %s...", cookies[:100])
		} else {
			logrus.Debugf("GetCookies: COOKIES 内容: %s", cookies)
		}
		return []byte(cookies), nil
	}

	logrus.Info("GetCookies: 环境变量 COOKIES 为空，尝试从文件获取")

	// 优先级 2-4：从文件获取
	path := GetCookiesFilePath()
	logrus.Infof("GetCookies: 尝试从文件获取，路径=%s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("GetCookies: 从文件读取失败: %v", err)
		return nil, errors.Wrap(err, "failed to read cookies from file")
	}

	logrus.Infof("GetCookies: 从文件获取成功，长度=%d", len(data))
	return data, nil
}

// GetCookiesFilePath 获取 cookies 文件路径。
// 为了向后兼容，如果旧路径 /tmp/cookies.json 存在，则继续使用；
// 否则使用当前目录下的 cookies.json
func GetCookiesFilePath() string {
	// 优先级 2：环境变量 COOKIES_PATH 指定的文件路径
	path := os.Getenv("COOKIES_PATH")
	if path != "" {
		return path
	}

	// 优先级 3：旧路径 /tmp/cookies.json（向后兼容）
	tmpDir := os.TempDir()
	oldPath := filepath.Join(tmpDir, "cookies.json")

	// 检查旧路径文件是否存在
	if _, err := os.Stat(oldPath); err == nil {
		// 文件存在，使用旧路径（向后兼容）
		return oldPath
	}

	// 优先级 4：当前目录下的 cookies.json
	return "cookies.json"
}