# SunnyNet MCP Server

<div align="center">

![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![MCP](https://img.shields.io/badge/MCP-Server-orange)

**SunnyNet 网络抓包工具的 MCP Server 实现**

支持通过 Model Context Protocol (MCP) 在 AI 应用中使用 SunnyNet 网络分析功能

[快速开始](#快速开始) • [功能说明](#功能说明) • [配置方法](#配置方法) • [使用示例](#使用示例)

</div>

---

## 界面预览

<div align="center">
<img src="./img/mcp-preview.png" alt="MCP Server 界面" width="800"/>
</div>

## 什么是 MCP Server

MCP (Model Context Protocol) 是 Anthropic 开发的协议，允许 AI 应用（如 Claude Desktop、Cursor 等）通过标准化接口调用外部工具和服务。

SunnyNet MCP Server 将强大的网络抓包功能集成到 AI 工作流中，让您可以通过对话的方式进行网络分析。

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

## 项目结构

```
sunnymcptool/
├── build/
│   └── bin/
│       └── SunnyNet.exe    # 预编译的 MCP Server（推荐使用）
├── mcp_server.go           # MCP Server 主程序
├── mcp_tools.go            # MCP 工具实现
├── mcp_stdio.go            # MCP 标准输入输出处理
├── mcp_standalone/         # 独立 MCP Server 版本
├── SunnyNet/               # SunnyNet 核心库
└── README.md               # 本文档
```

## 技术支持

- **QQ 交流群**: 277869228
- **Issues**: [GitHub Issues](https://github.com/a121400/sunnymcptool/issues)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 致谢

- [SunnyNet](https://github.com/qtgolang/SunnyNet) - 网络中间件核心库
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP 协议

---

<div align="center">
如果这个项目对您有帮助，请给一个 ⭐ Star 支持一下！
</div>

