# SunnyNet MCP Server

<div align="center">

![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![MCP](https://img.shields.io/badge/MCP-Server-orange)

**基于 SunnyNet 中间件和 SunnyNetTools 抓包工具扩展的 MCP Server**

在 SunnyNet 强大的网络分析能力基础上，通过 Model Context Protocol (MCP) 让 AI 应用能够调用网络抓包功能

[快速开始](#快速开始) • [功能说明](#功能说明) • [配置方法](#配置方法) • [使用示例](#使用示例)

</div>

---

## 项目说明

### 基于开源项目

本项目基于以下开源项目开发：

- **网络中间件核心**：[SunnyNet](https://github.com/qtgolang/SunnyNet) by [@qtgolang](https://github.com/qtgolang)
- **GUI 界面程序**：[SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) by [@qtgolang](https://github.com/qtgolang)

本项目在 SunnyNetTools 的基础上，添加了 **MCP (Model Context Protocol)** 支持，使得 SunnyNet 强大的网络分析功能可以在 AI 应用（如 Cursor、Claude Desktop）中使用。

> **重要声明**：GUI 界面代码来源于 [qtgolang/SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) 开源项目，我们在此基础上增加了 MCP Server 功能。感谢 [@qtgolang](https://github.com/qtgolang) 的卓越贡献和开源精神！

### 本项目的贡献

- [成功] 完整的 MCP Server 实现
- [成功] 15+ MCP Tools 工具集
- [成功] Cursor 和 Claude Desktop 集成支持
- [成功] MCP 协议标准化接口
- [成功] AI 工作流集成文档

## 界面预览

<div align="center">
<img src="./img/mcp-preview.png" alt="MCP Server 界面" width="800"/>
</div>

## 什么是 MCP Server

MCP (Model Context Protocol) 是 Anthropic 开发的协议，允许 AI 应用（如 Claude Desktop、Cursor 等）通过标准化接口调用外部工具和服务。

SunnyNet MCP Server 将 [SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) 的强大网络抓包功能集成到 AI 工作流中，让您可以通过对话的方式进行网络分析。

## 功能说明

### 核心功能

- [成功] **HTTP/HTTPS 抓包**：捕获和分析 HTTP/HTTPS 请求和响应
- [成功] **WebSocket 支持**：支持 WS/WSS 协议分析
- [成功] **TCP/UDP 抓包**：支持底层网络协议分析
- [成功] **数据解码**：自动解码 gzip、deflate、br、zstd 等压缩格式
- [成功] **进程过滤**：按进程名称过滤网络流量
- [成功] **证书管理**：自动管理 HTTPS 抓包证书
- [成功] **MCP 集成**：完整的 MCP 协议支持，可在 Cursor、Claude Desktop 等 AI 应用中使用

## 快速开始

### 方式一：使用预编译版本（推荐）

1. 下载预编译版本：
   - 位置：`build/bin/SunnyNet.exe`（Windows）
   - 或从 Releases 页面下载

2. 直接运行可执行文件启动 MCP Server

### 方式二：从源码编译

```bash
# 克隆仓库
git clone https://github.com/a121400/sunnymcptool.git
cd sunnymcptool

# 编译 MCP Server
go build -o sunnynet-mcp.exe mcp_server.go mcp_tools.go mcp_stdio.go

# 运行
./sunnynet-mcp.exe
```

## 配置方法

### 在 Cursor 中配置

1. 打开 Cursor 设置
2. 找到 MCP Servers 配置
3. 添加 SunnyNet MCP Server：

```json
{
  "mcpServers": {
    "sunnynet": {
      "command": "C:\\path\\to\\build\\bin\\SunnyNet.exe",
      "args": [],
      "env": {}
    }
  }
}
```

### 在 Claude Desktop 中配置

编辑 `%APPDATA%\Claude\claude_desktop_config.json`：

```json
{
  "mcpServers": {
    "sunnynet": {
      "command": "C:\\path\\to\\build\\bin\\SunnyNet.exe",
      "args": []
    }
  }
}
```

### 配置说明

- **command**: MCP Server 可执行文件的完整路径
- **args**: 启动参数（可选）
- **env**: 环境变量（可选）

## 使用示例

配置完成后，在 Cursor 或 Claude Desktop 中可以通过对话使用 SunnyNet 的功能：

### 示例 1：开始抓包

```
"启动 SunnyNet 抓包，监听端口 8888"
```

AI 将调用 MCP Server 启动抓包服务。

### 示例 2：查看请求

```
"显示最近捕获的 HTTP 请求"
```

AI 将返回最近的网络请求列表。

### 示例 3：分析特定请求

```
"分析请求 ID 为 12345 的详细信息"
```

AI 将显示该请求的完整头信息、响应内容等。

### 示例 4：过滤进程

```
"只抓取 chrome.exe 进程的网络流量"
```

AI 将配置进程过滤规则。

## MCP 功能详解

### 可用的 MCP Tools

SunnyNet MCP Server 提供以下完整功能工具：

#### 1. `sunnynet_start` - 启动网络抓包服务

启动 SunnyNet 网络抓包服务，开始监听和捕获网络流量。

**参数：**
- `port` (可选): 代理端口，默认 9999
- `enable_https` (可选): 是否启用 HTTPS 抓包，默认 true
- `driver` (可选): 网络驱动类型（netfilter/proxifier/tun），默认 netfilter

**返回：**
```json
{
  "success": true,
  "message": "SunnyNet 已启动",
  "port": 9999,
  "proxy_url": "http://127.0.0.1:9999"
}
```

**使用场景：**
- 开始新的抓包会话
- 配置特定端口和驱动
- 初始化 HTTPS 抓包环境

---

#### 2. `sunnynet_stop` - 停止抓包服务

停止当前运行的 SunnyNet 抓包服务。

**参数：** 无

**返回：**
```json
{
  "success": true,
  "message": "SunnyNet 已停止",
  "captured_requests": 1234
}
```

**使用场景：**
- 结束抓包会话
- 释放系统资源
- 导出数据前停止服务

---

#### 3. `sunnynet_get_requests` - 获取请求列表

获取已捕获的网络请求列表，支持分页和过滤。

**参数：**
- `limit` (可选): 返回数量，默认 50，最大 1000
- `offset` (可选): 偏移量，默认 0
- `protocol` (可选): 协议过滤（http/https/ws/wss/tcp/udp）
- `status_code` (可选): HTTP 状态码过滤（如 200, 404）
- `method` (可选): HTTP 方法过滤（GET/POST/PUT/DELETE等）
- `url_contains` (可选): URL 包含的关键字
- `process_name` (可选): 进程名称过滤

**返回：**
```json
{
  "success": true,
  "total": 1234,
  "requests": [
    {
      "id": "req_001",
      "method": "GET",
      "url": "https://api.example.com/users",
      "status": 200,
      "protocol": "https",
      "process": "chrome.exe",
      "timestamp": "2026-02-02T10:30:00Z",
      "size": 1024
    }
  ]
}
```

**使用场景：**
- 查看所有捕获的请求
- 按条件筛选特定请求
- 分析请求模式和趋势

---

#### 4. `sunnynet_get_request_detail` - 获取请求详细信息

获取指定请求的完整详细信息，包括请求头、响应头、Body 等。

**参数：**
- `request_id` (必需): 请求 ID

**返回：**
```json
{
  "success": true,
  "request": {
    "id": "req_001",
    "method": "POST",
    "url": "https://api.example.com/login",
    "protocol": "https",
    "status_code": 200,
    "request_headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token..."
    },
    "request_body": "{\"username\":\"test\"}",
    "response_headers": {
      "Content-Type": "application/json"
    },
    "response_body": "{\"status\":\"success\"}",
    "timing": {
      "dns": 10,
      "connect": 50,
      "ssl": 100,
      "send": 5,
      "wait": 200,
      "receive": 50,
      "total": 415
    }
  }
}
```

**使用场景：**
- 分析特定请求的完整信息
- 调试 API 调用问题
- 查看加密数据的解密内容

---

#### 5. `sunnynet_filter_process` - 设置进程过滤

配置只抓取特定进程的网络流量。

**参数：**
- `process_names` (必需): 进程名称数组，如 ["chrome.exe", "firefox.exe"]
- `mode` (可选): 过滤模式（whitelist/blacklist），默认 whitelist

**返回：**
```json
{
  "success": true,
  "message": "进程过滤已设置",
  "filtered_processes": ["chrome.exe", "firefox.exe"]
}
```

**使用场景：**
- 只监控特定应用的流量
- 排除系统进程干扰
- 精准定位目标程序

---

#### 6. `sunnynet_install_cert` - 安装 HTTPS 证书

安装或导出 SunnyNet 的 HTTPS 抓包证书。

**参数：**
- `action` (必需): 操作类型（install/export）
- `export_path` (可选): 导出路径（当 action=export 时）

**返回：**
```json
{
  "success": true,
  "message": "证书已安装到系统",
  "cert_path": "C:\\Users\\...\\sunnynet-ca.crt"
}
```

**使用场景：**
- 首次使用 HTTPS 抓包前安装证书
- 导出证书用于其他设备
- 重新安装损坏的证书

---

#### 7. `sunnynet_get_statistics` - 获取统计信息

获取当前会话的抓包统计数据。

**参数：** 无

**返回：**
```json
{
  "success": true,
  "statistics": {
    "total_requests": 1234,
    "by_protocol": {
      "http": 800,
      "https": 400,
      "ws": 20,
      "tcp": 14
    },
    "by_status": {
      "2xx": 1000,
      "3xx": 100,
      "4xx": 100,
      "5xx": 34
    },
    "total_size": 102400000,
    "uptime": 3600,
    "requests_per_second": 0.34
  }
}
```

**使用场景：**
- 查看抓包会话总览
- 分析流量分布
- 性能监控

---

#### 8. `sunnynet_modify_request` - 修改请求/响应

动态修改请求或响应数据（高级功能）。

**参数：**
- `request_id` (必需): 请求 ID
- `modify_type` (必需): 修改类型（request/response）
- `headers` (可选): 修改的头部
- `body` (可选): 修改的 Body
- `status_code` (可选): 修改的状态码（仅响应）

**返回：**
```json
{
  "success": true,
  "message": "请求已修改"
}
```

**使用场景：**
- 测试不同的请求参数
- 模拟服务器响应
- 调试客户端行为

---

#### 9. `sunnynet_export_data` - 导出抓包数据

将捕获的数据导出为文件（JSON/HAR 格式）。

**参数：**
- `format` (必需): 导出格式（json/har）
- `output_path` (必需): 输出文件路径
- `filter` (可选): 过滤条件（同 get_requests）

**返回：**
```json
{
  "success": true,
  "message": "数据已导出",
  "file_path": "C:\\exports\\capture_20260202.json",
  "exported_count": 1234
}
```

**使用场景：**
- 保存抓包结果
- 分享数据给团队
- 后续离线分析

---

#### 10. `sunnynet_websocket_monitor` - WebSocket 实时监控

监控 WebSocket 连接的实时消息。

**参数：**
- `connection_id` (必需): WebSocket 连接 ID
- `direction` (可选): 方向过滤（send/receive/both），默认 both

**返回：**
```json
{
  "success": true,
  "messages": [
    {
      "id": "msg_001",
      "direction": "send",
      "timestamp": "2026-02-02T10:30:01Z",
      "data": "{\"type\":\"ping\"}",
      "size": 15
    }
  ]
}
```

**使用场景：**
- 调试 WebSocket 通信
- 监控实时数据流
- 分析双向通信协议

---

#### 11. `sunnynet_set_replace_rules` - 设置替换规则

配置请求/响应内容的自动替换规则，实现数据拦截和修改。

**参数：**
- `rules` (必需): 替换规则数组
  - `type` (必需): 替换类型（String(UTF8)/String(GBK)/Base64/HEX/响应文件）
  - `source` (必需): 源内容（要被替换的内容）
  - `target` (必需): 目标内容（替换后的内容）
  - `hash` (可选): 规则唯一标识

**返回：**
```json
{
  "success": true,
  "message": "替换规则已设置",
  "total_rules": 5,
  "failed_rules": []
}
```

**替换类型说明：**
- **String(UTF8)**: UTF-8 编码的字符串替换
- **String(GBK)**: GBK 编码的字符串替换
- **Base64**: Base64 编码的数据替换
- **HEX**: 十六进制数据替换
- **响应文件**: 用指定文件内容替换响应

**示例配置：**
```json
{
  "rules": [
    {
      "type": "String(UTF8)",
      "source": "old_api_key",
      "target": "new_api_key",
      "hash": "rule_001"
    },
    {
      "type": "Base64",
      "source": "b2xkX2RhdGE=",
      "target": "bmV3X2RhdGE=",
      "hash": "rule_002"
    },
    {
      "type": "响应文件",
      "source": "https://api.example.com/data",
      "target": "C:\\mock\\response.json",
      "hash": "rule_003"
    }
  ]
}
```

**使用场景：**
- API 接口测试和模拟
- 修改加密参数
- 替换服务器响应内容
- 本地化调试
- 数据脱敏和替换

---

#### 12. `sunnynet_set_hosts_rules` - 设置 HOSTS 规则

配置域名重定向规则，实现自定义 DNS 解析和域名映射。

**参数：**
- `rules` (必需): HOSTS 规则数组
  - `source` (必需): 源域名（支持通配符 *）
  - `target` (必需): 目标地址
  - `hash` (可选): 规则唯一标识

**返回：**
```json
{
  "success": true,
  "message": "HOSTS 规则已设置",
  "total_rules": 3,
  "failed_rules": []
}
```

**通配符支持：**
- `*` : 匹配任意字符
- `*.example.com` : 匹配所有子域名
- `api.*.com` : 匹配中间部分

**示例配置：**
```json
{
  "rules": [
    {
      "source": "api.example.com",
      "target": "127.0.0.1:8080",
      "hash": "hosts_001"
    },
    {
      "source": "*.test.com",
      "target": "192.168.1.100",
      "hash": "hosts_002"
    },
    {
      "source": "old-domain.com",
      "target": "new-domain.com",
      "hash": "hosts_003"
    }
  ]
}
```

**使用场景：**
- 本地开发环境调试
- 测试环境域名映射
- 服务器迁移测试
- API 网关切换
- 域名替换和重定向

---

#### 13. `sunnynet_get_replace_rules` - 获取当前替换规则

获取当前生效的替换规则列表。

**参数：** 无

**返回：**
```json
{
  "success": true,
  "rules": [
    {
      "type": "String(UTF8)",
      "source": "old_api_key",
      "target": "new_api_key",
      "hash": "rule_001"
    }
  ],
  "total": 1
}
```

**使用场景：**
- 查看当前配置
- 规则审计
- 导出配置

---

#### 14. `sunnynet_get_hosts_rules` - 获取当前 HOSTS 规则

获取当前生效的 HOSTS 规则列表。

**参数：** 无

**返回：**
```json
{
  "success": true,
  "rules": [
    {
      "source": "api.example.com",
      "target": "127.0.0.1:8080",
      "hash": "hosts_001"
    }
  ],
  "total": 1
}
```

**使用场景：**
- 查看当前配置
- 域名映射管理
- 规则导出备份

---

#### 15. `sunnynet_clear_rules` - 清除规则

清除指定类型的规则或全部规则。

**参数：**
- `type` (必需): 规则类型（replace/hosts/all）

**返回：**
```json
{
  "success": true,
  "message": "规则已清除",
  "cleared_count": 5
}
```

**使用场景：**
- 重置配置
- 清理测试规则
- 切换场景配置

## 项目结构

```
sunnymcptool/
├── build/
│   └── bin/
│       └── SunnyNet.exe    # 预编译的 MCP Server（推荐使用）
├── frontend/               # Vue 3 前端界面
│   ├── src/
│   │   ├── components/    # Vue 组件
│   │   └── main.js        # 入口文件
│   └── package.json
├── SunnyNet/               # SunnyNet 核心库（子模块）
├── mcp_server.go           # MCP Server 主程序
├── mcp_tools.go            # MCP 工具实现
├── mcp_stdio.go            # MCP 标准输入输出处理
├── mcp_standalone/         # 独立 MCP Server 版本
├── go.mod                  # Go 模块配置
├── wails.json              # Wails 配置
└── README.md               # 本文档
```

## 开源技术栈

本项目基于以下优秀的开源项目构建：

### 核心框架

#### 网络中间件
- **[SunnyNet](https://github.com/qtgolang/SunnyNet)** `v1.0.0`
  - 作者：qtgolang
  - 功能：网络抓包和分析的核心引擎
  - 许可证：开源项目
  - 说明：提供 HTTP/HTTPS/WebSocket/TCP/UDP 全协议支持

#### 桌面应用框架
- **[Wails](https://github.com/wailsapp/wails)** `v2.11.0`
  - 作者：Wails 团队
  - 功能：Go + Web 技术构建桌面应用
  - 许可证：MIT
  - 说明：用于构建跨平台桌面 GUI 界面

#### 前端框架
- **[Vue 3](https://github.com/vuejs/core)** `v3.x`
  - 作者：Vue 团队
  - 功能：渐进式 JavaScript 框架
  - 许可证：MIT
  - 说明：用于构建现代化的用户界面

### 主要依赖库

#### Go 语言库

- **[gorilla/websocket](https://github.com/gorilla/websocket)** `v1.5.3`
  - WebSocket 协议实现

- **[traefik/yaegi](https://github.com/traefik/yaegi)** `v0.15.1`
  - Go 脚本解释器，支持动态脚本执行

- **[andybalholm/brotli](https://github.com/andybalholm/brotli)** `v1.1.1`
  - Brotli 压缩算法支持

- **[klauspost/compress](https://github.com/klauspost/compress)** `v1.17.11`
  - 高性能压缩库

- **[Trisia/gosysproxy](https://github.com/Trisia/gosysproxy)** `v1.1.0`
  - 系统代理设置工具

- **[mitchellh/go-ps](https://github.com/mitchellh/go-ps)** `v1.0.0`
  - 进程列表获取

- **[atotto/clipboard](https://github.com/atotto/clipboard)** `v0.1.4`
  - 跨平台剪贴板操作

- **[tidwall/gjson](https://github.com/tidwall/gjson)** `v1.18.0`
  - 快速 JSON 解析

- **[google/uuid](https://github.com/google/uuid)** `v1.6.0`
  - UUID 生成

- **[labstack/echo](https://github.com/labstack/echo)** `v4.13.3`
  - 高性能 Web 框架

### 前端组件库

- **Monaco Editor** - 代码编辑器
- **Element Plus** - Vue 3 UI 组件库
- **Vite** - 前端构建工具

## 技术支持

- **QQ 交流群**: 277869228
- **Issues**: [GitHub Issues](https://github.com/a121400/sunnymcptool/issues)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

所有依赖的开源项目遵循各自的开源许可证。

## 特别致谢

### 原作者及核心项目

本项目特别感谢 **[@qtgolang](https://github.com/qtgolang)** 及其开源贡献：

- **[SunnyNet](https://github.com/qtgolang/SunnyNet)** ![GitHub stars](https://img.shields.io/github/stars/qtgolang/SunnyNet?style=social)
  - 网络中间件核心引擎
  - 提供 HTTP/HTTPS/WebSocket/TCP/UDP 全协议支持
  - 本项目的技术基石
  
- **[SunnyNetTools](https://github.com/qtgolang/SunnyNetTools)** ![GitHub stars](https://img.shields.io/github/stars/qtgolang/SunnyNetTools?style=social)
  - **GUI 界面程序源代码**
  - 基于 Wails + Vue 构建的完整抓包工具
  - 本项目的 GUI 界面基于此项目开发

> **版权声明**：本项目的 GUI 界面代码来源于 [qtgolang/SunnyNetTools](https://github.com/qtgolang/SunnyNetTools)，我们在其基础上增加了 MCP Server 功能。所有原始代码的版权归 [@qtgolang](https://github.com/qtgolang) 所有。

### 框架和工具

感谢以下优秀的开源框架：

- **[Wails Team](https://wails.io/)** - 优秀的 Go 桌面应用框架，让跨平台开发变得简单
- **[Vue.js Team](https://vuejs.org/)** - 强大的前端框架，提供流畅的用户体验
- **[Model Context Protocol](https://modelcontextprotocol.io/)** - Anthropic 开发的 MCP 协议

### 社区支持

- **[Go Team](https://go.dev/)** - Go 语言及其生态系统
- **所有开源贡献者** - 感谢所有为开源社区做出贡献的开发者

### 开源精神

本项目秉承开源精神，站在巨人的肩膀上。特别感谢 [@qtgolang](https://github.com/qtgolang) 的 SunnyNet 和 SunnyNetTools 项目，为我们提供了强大的技术基础。我们希望通过添加 MCP 支持，让这些优秀的工具能够在 AI 工作流中发挥更大的价值。

如果您觉得这些项目有帮助，请分别给它们 Star 支持：
- [SunnyNet](https://github.com/qtgolang/SunnyNet) ⭐
- [SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) ⭐

---

<div align="center">
如果这个项目对您有帮助，请给一个 ⭐ Star 支持一下！
</div>

