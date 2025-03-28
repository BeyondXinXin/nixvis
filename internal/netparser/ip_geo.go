package netparser

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/sirupsen/logrus"
)

//go:embed data/ip2region.xdb
var ipDataFiles embed.FS

var (
	ipSearcher  *xdb.Searcher
	vectorIndex []byte
	dbPath      = filepath.Join(util.DataDir, "ip2region.xdb")
)

// ExtractIPRegionDB 从嵌入的文件系统中提取 IP2Region 数据库
func ExtractIPRegionDB() (string, error) {
	// 确保数据目录存在
	if _, err := os.Stat(util.DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(util.DataDir, 0755); err != nil {
			return "", err
		}
	}

	// 目标文件路径
	dbPath := filepath.Join(util.DataDir, "ip2region.xdb")

	// 检查文件是否已存在
	if _, err := os.Stat(dbPath); err == nil {
		logrus.Info("IP2Region 数据库已存在，跳过提取")
		return dbPath, nil
	}

	// 从嵌入文件系统读取数据
	data, err := fs.ReadFile(ipDataFiles, "data/ip2region.xdb")
	if err != nil {
		return "", err
	}

	// 写入文件
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		return "", err
	}

	logrus.Info("IP2Region 数据库已成功提取")
	return dbPath, nil
}

// InitIPGeoLocation 初始化 IP 地理位置查询
func InitIPGeoLocation() error {
	// 从嵌入的文件系统中提取数据库文件
	extractedPath, err := ExtractIPRegionDB()
	if err != nil {
		return fmt.Errorf("提取 ip2region 数据库失败: %v", err)
	}

	// 更新数据库路径
	dbPath = extractedPath

	// 加载矢量索引以加速搜索
	vIndex, err := xdb.LoadVectorIndexFromFile(dbPath)
	if err != nil {
		logrus.Warnf("加载 ip2region 矢量索引失败，将使用全量搜索: %v", err)
	} else {
		vectorIndex = vIndex
	}

	// 创建内存搜索器
	searcher, err := xdb.NewWithVectorIndex(dbPath, vectorIndex)
	if err != nil {
		return fmt.Errorf("创建 ip2region 搜索器失败: %v", err)
	}

	ipSearcher = searcher
	logrus.Info("ip2region 初始化成功")
	return nil
}

// GetIPLocation 获取 IP 的地理位置信息
func GetIPLocation(ip string) (string, string, error) {
	// 处理无效 IP
	if ip == "" || ip == "localhost" || ip == "127.0.0.1" {
		return "本地", "本地", nil
	}

	// 检查是否是内网 IP
	if isPrivateIP(net.ParseIP(ip)) {
		return "内网", "本地网络", nil
	}

	// 查询数据库
	domestic, global, err := queryIPLocation(ip)
	if err != nil {
		return "未知", "未知", err
	}

	return domestic, global, nil
}

// 查询 IP 地理位置
func queryIPLocation(ip string) (string, string, error) {
	if ipSearcher == nil {
		return "未知", "未知", fmt.Errorf("ip2region 未初始化")
	}

	// 设置 50 毫秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// 使用 channel 处理超时
	resultCh := make(chan struct {
		region string
		err    error
	}, 1)

	go func() {
		var region string
		var err error

		region, err = ipSearcher.SearchByStr(ip)

		resultCh <- struct {
			region string
			err    error
		}{region, err}
	}()

	// 等待结果或超时
	select {
	case <-ctx.Done():
		return "未知", "未知", fmt.Errorf("IP 查询超时")
	case result := <-resultCh:
		if result.err != nil {
			return "未知", "未知", result.err
		}
		return parseIPRegion(result.region)
	}
}

// 解析 ip2region 返回的地区信息
func parseIPRegion(region string) (string, string, error) {
	// 返回格式: 国家|区域|省份|城市|ISP
	parts := splitRegion(region)
	var domestic, global string

	// 国内
	if parts[0] == "中国" {
		if parts[2] != "" && parts[2] != "0" {
			domestic = removeSuffixes(parts[2])
		} else if parts[3] != "" && parts[3] != "0" {
			domestic = parts[3]
		} else {
			domestic = "中国"
		}
	} else if parts[0] == "0" || parts[0] == "" {
		domestic = "未知"
	} else {
		domestic = "国外"
	}

	// 全球
	if parts[0] != "0" && parts[0] != "" {
		global = parts[0]
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

	privateIPRanges := []struct {
		start net.IP
		end   net.IP
	}{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
	}

	for _, r := range privateIPRanges {
		if bytes.Compare(ip, r.start) >= 0 && bytes.Compare(ip, r.end) <= 0 {
			return true
		}
	}

	return false
}

// 去掉地区名称后缀
func removeSuffixes(name string) string {
	suffixes := []string{"省", "自治区", "维吾尔自治区", "壮族自治区", "回族自治区", "特别行政区"}
	for _, suffix := range suffixes {
		if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}
