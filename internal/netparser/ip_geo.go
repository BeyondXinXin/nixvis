package netparser

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/lionsoul2014/ip2region/binding/golang/service"
	"github.com/sirupsen/logrus"
)

//go:embed data/ip2region_v4.xdb
var ipDataFiles embed.FS

var (
	ipService *service.Ip2Region
	dbPath    = filepath.Join(util.DataDir, "ip2region_v4.xdb")
)

// ExtractIPRegionDB 从嵌入的文件系统中提取 IP2Region 数据库
func ExtractIPRegionDB() (string, error) {
	if err := os.MkdirAll(util.DataDir, 0755); err != nil {
		return "", err
	}

	dbPath := filepath.Join(util.DataDir, "ip2region_v4.xdb")

	// 读取 embed 内的最新 xdb
	data, err := fs.ReadFile(ipDataFiles, "data/ip2region_v4.xdb")
	if err != nil {
		return "", err
	}

	// 直接覆盖，避免 embed 更新后磁盘里还是旧文件
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		return "", err
	}

	logrus.Info("IP2Region v4 数据库已成功提取/更新")
	return dbPath, nil
}

// InitIPGeoLocation 初始化 IP 地理位置查询
func InitIPGeoLocation() error {
	extractedPath, err := ExtractIPRegionDB()
	if err != nil {
		return fmt.Errorf("提取 ip2region 数据库失败: %w", err)
	}

	dbPath = extractedPath

	// 只启用 v4；v6 传 nil
	v4Config, err := service.NewV4Config(service.VIndexCache, dbPath, 20)
	if err != nil {
		return fmt.Errorf("创建 ip2region v4 配置失败: %w", err)
	}

	svc, err := service.NewIp2Region(v4Config, nil)
	if err != nil {
		return fmt.Errorf("创建 ip2region 查询服务失败: %w", err)
	}

	ipService = svc
	logrus.Info("ip2region v4 初始化成功")
	return nil
}

// CloseIPGeoLocation 关闭查询服务（可选，但推荐在程序退出时调用）
func CloseIPGeoLocation() {
	if ipService != nil {
		ipService.Close()
		ipService = nil
	}
}

// GetIPLocation 获取 IP 的地理位置信息
func GetIPLocation(ip string) (string, string, error) {
	if ip == "" || ip == "localhost" {
		return "本地", "本地", nil
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "未知", "未知", fmt.Errorf("无效 IP: %s", ip)
	}

	if parsedIP.IsLoopback() {
		return "本地", "本地", nil
	}

	if isPrivateIP(parsedIP) {
		return "内网", "本地网络", nil
	}

	domestic, global, err := queryIPLocation(ip)
	if err != nil {
		return "未知", "未知", err
	}

	return domestic, global, nil
}

// 查询 IP 地理位置
func queryIPLocation(ip string) (string, string, error) {
	if ipService == nil {
		return "未知", "未知", fmt.Errorf("ip2region 未初始化")
	}

	region, err := ipService.SearchByStr(ip)
	if err != nil {
		return "未知", "未知", err
	}
	if region == "" {
		return "未知", "未知", nil
	}

	return parseIPRegion(region)
}

func parseIPRegion(region string) (string, string, error) {
	parts := splitRegion(region)
	var domestic, global string

	// 国内：只要省
	switch parts[0] {
	case "中国":
		if parts[1] != "" && parts[1] != "0" {
			domestic = removeSuffixes(parts[1])
		} else if parts[2] != "" && parts[2] != "0" {
			domestic = removeSuffixes(parts[2])
		} else {
			domestic = "中国"
		}
	case "0", "":
		domestic = "未知"
	default:
		domestic = "海外"
	}

	// 全球：只要国家
	if parts[0] != "0" && parts[0] != "" {
		global = translateCountryName(parts[0])
	} else {
		global = "未知"
	}

	return domestic, global, nil
}

// 解析 ip2region
func splitRegion(region string) []string {
	parts := make([]string, 5)
	fields := bytes.Split([]byte(region), []byte("|"))

	for i := 0; i < len(fields) && i < 5; i++ {
		parts[i] = string(fields[i])
	}

	return parts
}

// 是否是内网 IP
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// translateCountryName 将英文国家名翻译为中文，中文名直接返回
func translateCountryName(name string) string {
	if name == "" || name == "0" {
		return "未知"
	}
	if zh, ok := countryNameMap[name]; ok {
		return zh
	}
	// 已经是中文或未知英文名，原样返回
	return name
}

// countryNameMap 英文国家/地区名 -> 中文名 映射表
var countryNameMap = map[string]string{
	"中国":                     "中国",
	"United States":          "美国",
	"United Kingdom":         "英国",
	"Japan":                  "日本",
	"South Korea":            "韩国",
	"Germany":                "德国",
	"France":                 "法国",
	"Canada":                 "加拿大",
	"Australia":              "澳大利亚",
	"Russia":                 "俄罗斯",
	"India":                  "印度",
	"Brazil":                 "巴西",
	"Italy":                  "意大利",
	"Spain":                  "西班牙",
	"Netherlands":            "荷兰",
	"Singapore":              "新加坡",
	"Thailand":               "泰国",
	"Vietnam":                "越南",
	"Malaysia":               "马来西亚",
	"Indonesia":              "印度尼西亚",
	"Philippines":            "菲律宾",
	"Turkey":                 "土耳其",
	"Saudi Arabia":           "沙特阿拉伯",
	"United Arab Emirates":   "阿联酋",
	"Mexico":                 "墨西哥",
	"Argentina":              "阿根廷",
	"South Africa":           "南非",
	"Egypt":                  "埃及",
	"Nigeria":                "尼日利亚",
	"Israel":                 "以色列",
	"Sweden":                 "瑞典",
	"Norway":                 "挪威",
	"Denmark":                "丹麦",
	"Finland":                "芬兰",
	"Poland":                 "波兰",
	"Switzerland":            "瑞士",
	"Austria":                "奥地利",
	"Belgium":                "比利时",
	"Portugal":               "葡萄牙",
	"Ireland":                "爱尔兰",
	"New Zealand":            "新西兰",
	"Ukraine":                "乌克兰",
	"Czech Republic":         "捷克",
	"Romania":                "罗马尼亚",
	"Hungary":                "匈牙利",
	"Greece":                 "希腊",
	"Chile":                  "智利",
	"Colombia":               "哥伦比亚",
	"Peru":                   "秘鲁",
	"Venezuela":              "委内瑞拉",
	"Pakistan":               "巴基斯坦",
	"Bangladesh":             "孟加拉国",
	"Sri Lanka":              "斯里兰卡",
	"Myanmar":                "缅甸",
	"Cambodia":               "柬埔寨",
	"Laos":                   "老挝",
	"Mongolia":               "蒙古",
	"North Korea":            "朝鲜",
	"Nepal":                  "尼泊尔",
	"Iran":                   "伊朗",
	"Iraq":                   "伊拉克",
	"Kenya":                  "肯尼亚",
	"Morocco":                "摩洛哥",
	"Algeria":                "阿尔及利亚",
	"Tunisia":                "突尼斯",
	"Cuba":                   "古巴",
	"Luxembourg":             "卢森堡",
	"Iceland":                "冰岛",
	"Croatia":                "克罗地亚",
	"Serbia":                 "塞尔维亚",
	"Bulgaria":               "保加利亚",
	"Slovakia":               "斯洛伐克",
	"Slovenia":               "斯洛文尼亚",
	"Lithuania":              "立陶宛",
	"Latvia":                 "拉脱维亚",
	"Estonia":                "爱沙尼亚",
	"Kazakhstan":             "哈萨克斯坦",
	"Uzbekistan":             "乌兹别克斯坦",
	"Georgia":                "格鲁吉亚",
	"Azerbaijan":             "阿塞拜疆",
	"Armenia":                "亚美尼亚",
	"Qatar":                  "卡塔尔",
	"Kuwait":                 "科威特",
	"Bahrain":                "巴林",
	"Oman":                   "阿曼",
	"Jordan":                 "约旦",
	"Lebanon":                "黎巴嫩",
	"Cyprus":                 "塞浦路斯",
	"Malta":                  "马耳他",
	"Panama":                 "巴拿马",
	"Costa Rica":             "哥斯达黎加",
	"Ecuador":                "厄瓜多尔",
	"Bolivia":                "玻利维亚",
	"Paraguay":               "巴拉圭",
	"Uruguay":                "乌拉圭",
	"Dominican Republic":     "多米尼加",
	"Guatemala":              "危地马拉",
	"Honduras":               "洪都拉斯",
	"El Salvador":            "萨尔瓦多",
	"Jamaica":                "牙买加",
	"Trinidad and Tobago":    "特立尼达和多巴哥",
	"Ethiopia":               "埃塞俄比亚",
	"Tanzania":               "坦桑尼亚",
	"Ghana":                  "加纳",
	"Uganda":                 "乌干达",
	"Mozambique":             "莫桑比克",
	"Angola":                 "安哥拉",
	"Cameroon":               "喀麦隆",
	"Senegal":                "塞内加尔",
	"Madagascar":             "马达加斯加",
	"DR Congo":               "刚果(金)",
	"Afghanistan":            "阿富汗",
	"Turkmenistan":           "土库曼斯坦",
	"Kyrgyzstan":             "吉尔吉斯斯坦",
	"Tajikistan":             "塔吉克斯坦",
	"Belarus":                "白俄罗斯",
	"Moldova":                "摩尔多瓦",
	"North Macedonia":        "北马其顿",
	"Albania":                "阿尔巴尼亚",
	"Montenegro":             "黑山",
	"Bosnia and Herzegovina": "波黑",
	"Kosovo":                 "科索沃",
	"Taiwan":                 "台湾",
	"Hong Kong":              "香港",
	"Macao":                  "澳门",
}

// 去掉地区名称后缀
func removeSuffixes(name string) string {
	suffixes := []string{
		"维吾尔自治区",
		"壮族自治区",
		"回族自治区",
		"特别行政区",
		"自治区",
		"省",
		"市",
	}
	for _, suffix := range suffixes {
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}
