package moviesearch

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	DefaultJavDBDomain  = "www.javdb.com"
	DefaultJavBusDomain = "www.javbus.com"
)

// NormalizeDomain 校验并规范化不含协议和路径的域名。
func NormalizeDomain(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("域名不能为空")
	}
	if strings.Contains(value, "://") {
		return "", fmt.Errorf("域名不能包含协议")
	}

	parsed, err := url.Parse("https://" + value)
	if err != nil || parsed.Host == "" || parsed.Hostname() == "" {
		return "", fmt.Errorf("域名格式无效")
	}
	if parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host != value {
		return "", fmt.Errorf("域名不能包含路径、参数或认证信息")
	}
	return value, nil
}

func baseURL(domain string) string {
	return "https://" + domain
}
