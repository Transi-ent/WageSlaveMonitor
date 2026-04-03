# WageSlaveMonitor

[English README](README-EN.md)

## 简介

作为牛马打工人，有没有被公司的监控软件困扰过？所谓 "知己知己方可百战不殆"，该项目就是一个这样的软件，取名为 WageSlaveMonitor。

WageSlaveMonitor 的实现是一套轻量级的 **Windows 客户端 + Linux 服务端** 方案。Windows 端按周期截取 **当前连接的全部显示器** 画面（JPEG），将截图上传至服务端；**断网时** 写入本地队列，网络恢复后继续补传。服务端将图片存于磁盘、元数据存 **SQLite**，提供 **HTTP API** 与 **Web 控制台**，按客户端分组浏览、按时间排序查看截图，并支持从服务端侧调整 **截屏周期**。

**重要声明：** 仅在具备合法授权与用户知情同意的前提下部署使用。禁止用于隐蔽监控、未授权采集或其他违法用途。

## 功能与特色

| 类别 | 说明 |
|------|------|
| **多显示器截屏** | 在 Windows 上一次周期内截取所有活动显示器（JPEG，质量可在代码中调节）。 |
| **远程调整周期** | 服务端按客户端保存 `capture_interval_seconds`，客户端轮询 `GET /api/v1/clients/{id}/config`。 |
| **断网缓冲** | 上传失败时写入 `AGENT_DATA_DIR/spool` 本地文件队列，联网后自动继续上传。 |
| **服务端轻量** | 单一 Go 可执行文件：SQLite 索引 + 文件系统存图；可选按天保留策略（`RETENTION_DAYS`）。 |
| **Web 控制台** | 客户端列表、按时间线查看（新到旧）、图片预览；表单修改截屏间隔。 |
| **控制台登录** | 首次启动默认密码 **`123456`**（bcrypt 存在 SQLite）；登录后可通过 **修改密码** 更改。Cookie 会话；若配置 `AUTH_TOKEN`，API 仍可使用 `Authorization: Bearer`。 |
| **Windows 服务** | 子命令 `install` / `uninstall` 通过 `sc.exe` 注册服务名 `WageSlaveMonitorAgent`（需管理员权限）。 |
| **运维** | `GET /healthz` 健康检查、请求日志；Linux 下 `systemd` 示例见 [docs/deployment-linux.md](docs/deployment-linux.md)。 |

架构与接口详见 [docs/mvp-architecture.md](docs/mvp-architecture.md)。

## 快速开始

### 环境要求

- **Go** 1.21+（具体以仓库 `go.mod` 为准）。
- **服务端：** 生产环境建议 **Linux**；本地开发可在任意支持 Go 的系统运行。
- **客户端：** 仅 **Windows** 具备真实截屏能力（其他平台为占位实现）。

### 服务端（本地开发）

```bash
cd server
go run ./cmd/server
```

服务端配置从 `server/config/config.json` 读取（相对于 `server/` 目录）。

主要配置项：

| 配置项 | 说明 |
|--------|------|
| `ADDR` | 监听地址（默认 `:8080`）。 |
| `DATA_DIR` | 截图与数据库根目录（默认 `./data`）。 |
| `DB_PATH` | SQLite 文件路径（默认 `./data/meta.db`）。 |
| `AUTH_TOKEN` | 若设置，客户端上传等 API 需携带 `Authorization: Bearer <token>`。 |
| `DEFAULT_CAPTURE_INTERVAL_SECONDS` | 新客户端默认截屏间隔秒数（默认 `30`）。 |
| `RETENTION_DAYS` | 自动删除早于该天数的截图（默认 `14`）。 |
| `CONSOLE_AUTH_DISABLED` | 设为 `true` 则禁用控制台登录（默认 `true`，方便测试）。 |

如需启用密码保护，将配置文件中 `CONSOLE_AUTH_DISABLED` 改为 `false`，默认密码为 `123456`。

**Linux 生产部署**（编译、`systemd`、配置文件）：见 [docs/deployment-linux.md](docs/deployment-linux.md)。

### 客户端（Windows）

```powershell
cd client
$env:SERVER_BASE_URL = "http://你的服务器:8080"
$env:AUTH_TOKEN = "与服务端一致（若已设置）"
$env:AGENT_DATA_DIR = ".\agent-data"
go run .\cmd\agent
```

首次运行会在 `AGENT_DATA_DIR` 下生成稳定的 `client-id.txt`。

**安装为 Windows 服务（管理员权限）**

```powershell
cd client
go build -o WageSlaveAgent.exe .\cmd\agent
.\WageSlaveAgent.exe install
# 卸载：
.\WageSlaveAgent.exe uninstall
```

服务的运行环境变量（如 `SERVER_BASE_URL`、`AUTH_TOKEN`）需按你的部署方式配置（例如通过服务配置或 `sc.exe` 等）。

### Web 控制台

1. 浏览器打开 `http://你的服务器:8080/console/clients`。
2. 若服务端配置中 `CONSOLE_AUTH_DISABLED` 为 `false`，需使用初始密码 **`123456`** 登录，随后在顶部 **修改密码** 设为强密码（至少 6 位）。
3. 进入具体客户端页面可查看按时间排序的截图，并通过表单修改 **截屏间隔（秒）**。

**API（自动化可选）**

- 列出客户端：`GET /api/v1/clients`（若启用 `AUTH_TOKEN`，需 Bearer 头）。
- 上传与配置接口同样支持 Bearer；详见 [docs/mvp-architecture.md](docs/mvp-architecture.md)。

### 健康检查

```bash
curl http://127.0.0.1:8080/healthz
```

应返回正文 `ok`。

## 欢迎 Star / Fork

如果本项目对你有帮助，欢迎在 GitHub 上 **点个 Star**，方便更多人发现，也能给维护者一点动力。**Fork** 仓库可以自由二次开发与实验；欢迎提交 Issue 反馈问题或建议，也欢迎合规、透明用途下的 **Pull Request** 改进文档与代码。

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
