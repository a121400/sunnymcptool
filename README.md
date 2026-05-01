# SunnyNet MCP Server

基于 [SunnyNet](https://github.com/qtgolang/SunnyNet) + [SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) 的 MCP Server，让 AI 直接调用网络抓包功能。

## 快速开始

### 1. 下载

从 [Releases](https://github.com/a121400/sunnymcptool/releases) 下载对应平台：

**Windows:**
- `SunnyNet.exe` — GUI 主程序（含内置 MCP Server）
- `sunnynet-mcp.exe` — MCP stdio 桥接

**macOS (Apple Silicon):**
- `SunnyNet.app` — GUI 主程序（含内置 MCP Server）
- `sunnynet-mcp` — MCP stdio 桥接

### 2. 配置 Cursor

**Windows:**
```json
{
  "mcpServers": {
    "sunnynet": {
      "command": "C:\\path\\to\\sunnynet-mcp.exe",
      "args": ["-port", "29999"]
    }
  }
}
```

**macOS:**
```json
{
  "mcpServers": {
    "sunnynet": {
      "command": "/path/to/sunnynet-mcp",
      "args": ["-port", "29999"]
    }
  }
}
```

先启动 GUI 主程序（SunnyNet.exe 或 SunnyNet.app），MCP 桥接通过 HTTP 转发请求到主程序的 MCP 端点（默认 :29999/mcp）。

### 3. 使用

在 Cursor 中直接对话：
- "启动抓包"
- "显示最近的 HTTP 请求"
- "分析请求 12345 的详情"
- "只抓 chrome.exe 的流量"
- "搜索包含 login 的请求"

## MCP 工具列表

| 分类 | 工具 | 说明 |
|------|------|------|
| **代理控制** | `proxy_start` | 启动抓包 |
| | `proxy_stop` | 停止抓包 |
| | `proxy_set_port` | 设置代理端口 |
| | `proxy_get_status` | 获取代理状态 |
| **请求操作** | `request_list` | 获取请求列表 |
| | `request_get` | 获取请求详情 |
| | `request_search` | 搜索请求 |
| | `request_body_search` | 在 Body 中搜索关键词 |
| | `request_get_response_body_decoded` | 获取解码后的响应体 |
| | `request_modify_header` | 修改请求头 |
| | `request_modify_body` | 修改请求体 |
| | `response_modify_header` | 修改响应头 |
| | `response_modify_body` | 修改响应体 |
| | `request_block` | 阻断请求 |
| | `request_release_all` | 放行所有请求 |
| | `request_resend` | 重发请求 |
| | `request_delete` | 删除请求记录 |
| | `request_stats` | 抓包统计信息 |
| | `request_save_all` | 保存全部抓包数据 |
| | `request_import` | 导入抓包数据 |
| **长连接** | `connection_list` | 列出所有 WS/TCP/UDP 连接 |
| | `socket_data_list` | 连接的数据包列表 |
| | `socket_data_get` | 获取单个数据包详情 |
| | `socket_data_get_range` | 批量获取数据包 |
| **断点** | `breakpoint_add` | 添加断点条件 |
| | `breakpoint_list` | 列出断点条件 |
| | `breakpoint_remove` | 删除断点条件 |
| | `breakpoint_clear` | 清空断点 |
| **规则** | `replace_rules_list` | 列出替换规则 |
| | `replace_rules_add` | 添加替换规则 |
| | `replace_rules_remove` | 删除替换规则 |
| | `replace_rules_clear` | 清空替换规则 |
| | `hosts_rules_list` | 列出 HOSTS 规则 |
| | `hosts_rules_add` | 添加 HOSTS 规则 |
| | `hosts_rules_clear` | 清空 HOSTS 规则 |
| **进程** | `process_add` | 添加进程过滤 |
| | `process_remove` | 移除进程过滤 |
| | `process_list` | 列出已过滤进程 |
| **证书** | `cert_install` | 安装 HTTPS 证书 |
| | `cert_status` | 证书状态 |
| **配置** | `config_get` | 获取全部配置 |

## 从源码编译

**Windows (PowerShell):**
```powershell
git clone https://github.com/a121400/sunnymcptool.git
cd sunnymcptool

.\scripts\build.ps1              # 全部构建
.\scripts\build.ps1 -Target gui  # 仅 GUI
.\scripts\build.ps1 -Target mcp  # 仅 MCP
```

**macOS (需要 Go 1.21+ 和 Wails v2):**
```bash
git clone --recurse-submodules https://github.com/a121400/sunnymcptool.git
cd sunnymcptool

# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 编译 GUI
wails build -platform darwin/arm64

# 编译 MCP 桥接
cd headless && go build -o ../build/bin/sunnynet-mcp .
```

产物在 `build/bin/` 目录。

## iOS 抓包工具（越狱设备）

`ios-tool/` 目录包含：
- `hide_proxy.c` — CydiaSubstrate tweak，hook CFNetworkCopySystemProxySettings 绕过 App 代理检测
- `HideProxy.plist` — MobileSubstrate 过滤器
- `sunny-tool.sh` — 越狱 iOS 一键安装 CA 证书和配置 WiFi 代理的脚本

## 致谢

- [SunnyNet](https://github.com/qtgolang/SunnyNet) — 网络中间件核心 by [@qtgolang](https://github.com/qtgolang)
- [SunnyNetTools](https://github.com/qtgolang/SunnyNetTools) — GUI 界面 by [@qtgolang](https://github.com/qtgolang)

## License

MIT
