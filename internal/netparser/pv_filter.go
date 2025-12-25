package netparser

import (
	"regexp"

	"github.com/beyondxinxin/nixvis/internal/util"
)

var (
	// 全局过滤规则编译后的正则表达式
	excludePatterns []*regexp.Regexp
	excludeIPs      map[string]bool
	statusCodes     map[int]bool
)

// InitPVFilters 初始化PV过滤规则
func InitPVFilters() {
	cfg := util.ReadConfig()

	// 初始化状态码过滤
	statusCodes = make(map[int]bool)
	for _, code := range cfg.PVFilter.StatusCodeInclude {
		statusCodes[code] = true
	}

	// 初始化正则表达式过滤
	excludePatterns = make([]*regexp.Regexp, len(cfg.PVFilter.ExcludePatterns))
	for i, pattern := range cfg.PVFilter.ExcludePatterns {
		excludePatterns[i] = regexp.MustCompile(pattern)
	}

	// 初始化IP过滤
	excludeIPs = make(map[string]bool)
	for _, ip := range cfg.PVFilter.ExcludeIPs {
		excludeIPs[ip] = true
	}
}

// ShouldCountAsPageView 判断是否符合 PV 过滤条件
func ShouldCountAsPageView(statusCode int, path string, ip string) int {
	// 检查状态码
	if !statusCodes[statusCode] {
		return 0
	}

	// 检查排除 IP 列表
	if excludeIPs[ip] {
		return 0
	}

	// 检查是否匹配全局排除模式
	for _, pattern := range excludePatterns {
		if pattern.MatchString(path) {
			return 0
		}
	}

	return 1
}
