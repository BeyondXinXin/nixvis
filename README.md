# NixVis

![](https://github.com/BeyondXinXin/nixvis/actions/workflows/ci-linux.yml/badge.svg?branch=main)

NixVis 是一款基于 Go 语言开发的、开源轻量级 Nginx 日志分析工具，专为自部署场景设计。它提供直观的数据可视化和全面的统计分析功能，帮助您实时监控网站流量、访问来源和地理分布等关键指标，无需复杂配置即可快速部署使用。

演示地址 [nixvis.beyondxin](https://nixvis.beyondxin.top/)

![](https://img.beyondxin.top/2025/202503291600721.png)

## 功能特点

- **全面访问指标**：实时统计独立访客数 (UV)、页面浏览量 (PV) 和流量数据
- **地理位置分布**：展示国内和全球访问来源的可视化地图
- **详细访问排名**：提供 URL、引荐来源、浏览器、操作系统和设备类型的访问排名
- **时间序列分析**：支持按小时和按天查看访问趋势
- **多站点支持**：可同时监控多个网站的访问数据
- **增量日志解析**：自动扫描 Nginx 日志文件，解析并存储最新数据
- **高性能查询**：存储使用轻量级 SQLite，结合多级缓存策略实现快速响应
- **嵌入式资源**：前端资源和IP库内嵌于可执行文件中，无需额外部署静态文件

## 快速开始

1. 下载最新版本的 NixVis

```bash
wget https://github.com/beyondxinxin/nixvis/releases/download/latest/nixvis
chmod +x nixvis
```

2. 生成配置文件
```bash
./nixvis -gen-config
```
执行后将在当前目录生成 nixvis_config.json 配置文件。

3. 编辑配置文件 nixvis_config.json，添加您的网站信息和日志路径

支持自动切割的日志路径（详见 [说明](https://github.com/BeyondXinXin/nixvis/issues/2)）

```json
{
  "websites": [
    {
      "name": "示例网站1",
      "logPath": "./weblog_eg/blog.beyondxin.top.log"
    },
    {
      "name": "示例网站2",
      "logPath": "/var/log/nginx/blog.log"
    }
  ],
  "system": {
    "logDestination": "file",
    "taskInterval": "5m"
  },
  "server": {
    "Port": ":8088"
  },
  "pvFilter": {
    "statusCodeInclude": [
      200
    ],
    "excludePatterns": [
      "favicon.ico$",
      "robots.txt$",
      "sitemap.xml$",
      "\\.(?:js|css|jpg|jpeg|png|gif|svg|webp|woff|woff2|ttf|eot|ico)$",
      "^/api/",
      "^/ajax/",
      "^/health$",
      "^/_(?:nuxt|next)/",
      "rss.xml$",
      "feed.xml$",
      "atom.xml$"
    ]
  }
}
```

4. 启动 NixVis 服务
```bash
./nixvis
```

5. 访问 Web 界面
http://localhost:8088


## 从源码编译

如果您想从源码编译 NixVis，请按照以下步骤操作：

```bash
# 克隆项目仓库
git clone https://github.com/BeyondXinXin/nixvis.git
cd nixvis

# 编译项目
go mod tidy
go build -o nixvis ./cmd/nixvis/main.go

# 或使用编译脚本
# bash package.sh
```

## docker部署

1. 下载 docker-compose

```bash
wget https://github.com/beyondxinxin/nixvis/releases/download/docker/docker-compose.yml
wget https://github.com/beyondxinxin/nixvis/releases/download/docker/nixvis_config.json
```

2. 修改 nixvis_config.json 添加您的网站信息和日志路径

3. 修改 docker-compose.yml 添加文件挂载(nixvis_config.json、日志文件)

如需分析多个日志文件，可以考虑将日志目录整体挂载（如 /var/log/nginx:/var/log/nginx:ro）。

```yml
version: '3'
services:
  nixvis:
    image: ${{ secrets.DOCKERHUB_USERNAME }}/nixvis:latest
    ports:
      - "8088:8088"
    volumes:
      - ./nixvis_config.json:/app/nixvis_config.json:ro
      - /var/log/nginx/blog.log:/var/log/nginx/blog.log:ro
      - /etc/localtime:/etc/localtime:ro
```

4. 启动

```bash
docker compose up -d
```

5. 访问 Web 界面
http://localhost:8088

## 技术栈

- **后端**: Go语言 (Gin框架、ip2region地理位置查询)
- **前端**: 原生HTML5/CSS3/JavaScript (ECharts地图可视化、Chart.js图表)

## 许可证

NixVis 使用 MIT 许可证开源发布。详情请查看 LICENSE 文件。
