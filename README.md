# NixVis

一个轻量、自部署的 Nginx access log 统计工具。下载一个可执行文件或启动一个 Docker 容器，即可查看网站访问数据。

> NixVis 没有内置登录。请只在内网访问，或使用带认证的反向代理，不要直接暴露到公网。

## 快速开始

### Linux

从 [Releases](https://github.com/BeyondXinXin/nixvis/releases) 下载对应版本，或获取最新稳定版：

```bash
wget https://github.com/BeyondXinXin/nixvis/releases/latest/download/nixvis-linux-amd64
chmod +x nixvis-linux-amd64
mv nixvis-linux-amd64 nixvis
./nixvis -gen-config
```

编辑生成的 `nixvis_config.json`，将示例站点改为自己的 access log 文件：

```json
{
  "name": "我的网站",
  "logPath": "/var/log/nginx/access.log"
}
```

然后启动并访问 `http://localhost:8088`：

```bash
./nixvis
```

### Docker

下载同一版本发布的两个文件：

```bash
wget https://github.com/BeyondXinXin/nixvis/releases/latest/download/docker-compose.yml
wget https://github.com/BeyondXinXin/nixvis/releases/latest/download/nixvis_config.json
```

编辑 `nixvis_config.json` 中的站点和 `docker-compose.yml` 中的日志挂载路径，再启动：

```bash
docker compose up -d
```

统计数据保存在 `nixvis_data` 中。正常升级不需要删除它：

```bash
docker compose pull
docker compose up -d
```

## 配置注意事项

- `logPath` 填 access log 的**文件路径**，不是目录；轮转日志可使用 glob，例如 `/var/log/nginx/access.log*`。
- 不支持读取 `.gz` 压缩日志；请在 glob 中排除它们。
- 使用标准 Nginx access log（combined 格式）。自定义 `log_format` 不保证可解析。
- 程序每 5 分钟增量读取一次日志；可通过 `system.taskInterval` 调整，最小为 5 秒。
- 运行 `./nixvis -v` 可查看当前二进制版本、构建时间和提交号。

## 特点

- 单个可执行文件或 Docker 容器，无需额外服务
- 多站点与日志轮转支持
- PV、UV、流量、URL、来源、浏览器、设备与地域统计
- 小时和天维度的访问趋势
- 本地 SQLite 存储与增量读取
- 前端资源和 IP 地理库内嵌，无需部署静态文件

## 许可证

NixVis 使用 [MIT License](LICENSE) 开源发布。
