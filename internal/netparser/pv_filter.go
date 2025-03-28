package netparser

import "regexp"

// IsPageView 判断是否符合 PV 过滤条件
func IsPageView(statusCode int, path string) int {
	// PV 过滤条件：
	// 1. status = 200
	// 2. path 中不含 favicon、sitemap、rss、robots.txt
	// 3. 开头不含 /_nuxt

	if statusCode == 200 &&
		!regexp.MustCompile(`favicon|sitemap|rss|robots.txt`).MatchString(path) &&
		!regexp.MustCompile(`^/_nuxt`).MatchString(path) {
		return 1
	}
	return 0
}
