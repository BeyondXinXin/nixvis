package stats

import (
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"

	"github.com/beyondxinxin/nixvis/internal/util"
)

type refererDomainStats struct {
	pv  int
	ips map[string]struct{}
}

type refererDomainRank struct {
	domain string
	pv     int
	uv     int
}

func (s *ClientStatsManager) queryRefererStats(query StatsQuery, startUnix, endUnix int64, limit int) (StatsResult, error) {
	result := ClientStats{
		Key:       make([]string, 0),
		PV:        make([]int, 0),
		UV:        make([]int, 0),
		PVPercent: make([]int, 0),
		UVPercent: make([]int, 0),
	}

	dbQueryStr := fmt.Sprintf(`
        SELECT referer, ip
        FROM "%[1]s_nginx_logs" INDEXED BY idx_%[1]s_pv_ts_ip
        WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?
            AND length(referer) > 0
            AND referer <> char(45)`,
		query.WebsiteID)

	rows, err := s.repo.GetDB().Query(dbQueryStr, startUnix, endUnix)
	if err != nil {
		return result, fmt.Errorf("查询来源域名统计失败: %v", err)
	}
	defer rows.Close()

	internalDomain := currentWebsiteDomain(query.WebsiteID)
	domains := make(map[string]*refererDomainStats)

	for rows.Next() {
		var referer string
		var ip string
		if err := rows.Scan(&referer, &ip); err != nil {
			return result, fmt.Errorf("解析来源域名统计失败: %v", err)
		}

		domain, ok := normalizeRefererDomain(referer)
		if !ok || !looksLikeDomain(domain) || isInternalDomain(domain, internalDomain) {
			continue
		}

		item, ok := domains[domain]
		if !ok {
			item = &refererDomainStats{ips: make(map[string]struct{})}
			domains[domain] = item
		}
		item.pv++
		item.ips[ip] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历来源域名统计失败: %v", err)
	}

	ranks := make([]refererDomainRank, 0, len(domains))
	totalPV := 0
	totalUV := 0
	for domain, item := range domains {
		uv := len(item.ips)
		ranks = append(ranks, refererDomainRank{domain: domain, pv: item.pv, uv: uv})
		totalPV += item.pv
		totalUV += uv
	}

	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].uv != ranks[j].uv {
			return ranks[i].uv > ranks[j].uv
		}
		if ranks[i].pv != ranks[j].pv {
			return ranks[i].pv > ranks[j].pv
		}
		return ranks[i].domain < ranks[j].domain
	})

	if limit > len(ranks) {
		limit = len(ranks)
	}

	for i := 0; i < limit; i++ {
		result.Key = append(result.Key, ranks[i].domain)
		result.PV = append(result.PV, ranks[i].pv)
		result.UV = append(result.UV, ranks[i].uv)
	}

	if totalPV > 0 && totalUV > 0 {
		for i := range result.PV {
			result.PVPercent = append(result.PVPercent, int(math.Round(float64(result.PV[i])/float64(totalPV)*100)))
			result.UVPercent = append(result.UVPercent, int(math.Round(float64(result.UV[i])/float64(totalUV)*100)))
		}
	}

	return result, nil
}

func currentWebsiteDomain(websiteID string) string {
	website, ok := util.GetWebsiteByID(websiteID)
	if !ok {
		return ""
	}

	domain, ok := normalizeRefererDomain(website.Name)
	if !ok || !looksLikeDomain(domain) {
		return ""
	}
	return domain
}

func normalizeRefererDomain(referer string) (string, bool) {
	referer = strings.TrimSpace(referer)
	if referer == "" || referer == "-" || strings.EqualFold(referer, "about:blank") || strings.EqualFold(referer, "null") {
		return "", false
	}

	if strings.HasPrefix(referer, "/") && !strings.HasPrefix(referer, "//") {
		return "", false
	}
	if strings.HasPrefix(referer, "//") {
		referer = "http:" + referer
	}

	parsed, err := url.Parse(referer)
	if err != nil {
		return "", false
	}
	host := parsed.Hostname()

	if host == "" {
		parsed, err = url.Parse("http://" + referer)
		if err != nil {
			return "", false
		}
		host = parsed.Hostname()
	}

	host = strings.TrimSuffix(strings.ToLower(host), ".")
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return "", false
	}
	return host, true
}

func isInternalDomain(domain, internalDomain string) bool {
	if internalDomain == "" {
		return false
	}
	return domain == internalDomain || strings.HasSuffix(domain, "."+internalDomain)
}

func looksLikeDomain(domain string) bool {
	return domain == "localhost" || strings.Contains(domain, ".")
}
