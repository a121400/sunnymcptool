# SunnyNet MCP 工具

<div align="center">

![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Wails](https://img.shields.io/badge/Wails-v2.11.0-orange)
![Vue](https://img.shields.io/badge/Vue-3.x-brightgreen)

**功能强大的跨平台网络分析与抓包工具**

基于 Wails + Vue 构建，核心功能由 [SunnyNet](https://github.com/qtgolang/SunnyNet) 提供

[功能特性](#功能特性) • [快速开始](#快速开始) • [使用说明](#使用说明) • [编译构建](#编译构建) • [贡献指南](#贡献指南)

</div>

---

## 截图预览

<div align="center">
<img src="./img/1.jpg" alt="主界面" width="800"/>
<img src="./img/2.jpg" alt="功能界面" width="800"/>
</div>

## 功能特性

### 网络分析能力

- [成功] **多协议支持**：HTTP/HTTPS/WS/WSS/TCP/UDP 全协议网络分析
- [成功] **数据捕获修改**：可实时获取和修改所有协议的发送及返回数据
- [成功] **代理设置**：可为指定连接设置独立代理
- [成功] **连接重定向**：支持 HTTP/HTTPS/WS/WSS/TCP/TLS-TCP 链接重定向
- [成功] **数据解码**：支持 gzip、deflate、br、zstd 等多种压缩格式解码
- [成功] **主动发送**：支持 WS/WSS/TCP/TLS-TCP/UDP 主动发送数据

### 应用特性

- [成功] **跨平台支持**：Windows、Linux、macOS 全平台支持
- [成功] **现代化界面**：基于 Vue 3 的现代化用户界面
- [成功] **脚本支持**：支持通过 Go 脚本自定义处理逻辑
- [成功] **MCP 集成**：集成 Model Context Protocol 支持

### 多驱动支持

| 驱动名称 | 平台 | 127.0.0.1捕获 | 内网捕获 | 兼容性 |
|---------|------|-------------|---------|--------|
| Netfilter | Windows | 支持 | 支持 | 一般 |
| Proxifier | Windows | 支持 | 支持 | 一般 |
| Tun(WinDivert) | Windows | 不支持 | 支持 | 较好 |
| Tun(VPN) | Android | 支持 | 支持 | 较好 |
| Tun(utun) | MacOS | 不支持 | 不支持 | 较好 |
| Tun(tun) | Linux | 不支持 | 不支持 | 较好 |

## 快速开始

### 系统要求

- **Windows**: Windows 10/11（推荐）或 Windows 7+（需使用 Go 1.21 以下版本编译）
- **Linux**: 最新稳定版本
- **macOS**: 最新稳定版本
- **Go**: >= 1.23.0
- **Node.js**: >= 16.x

### 预编译版本下载

- **Windows 版本**
  - 下载地址: https://wwxa.lanzouj.com/b0cior9kb
  - 密码: 2brf

- **macOS 版本**
  - 下载地址: https://wwxa.lanzouj.com/b0ciopv1c
  - 密码: 2oxf

## 编译构建

### 前置要求

#### Windows 编译环境

1. 安装 [TDM-GCC](https://github.com/jmeubank/tdm-gcc/releases/download/v10.3.0-tdm64-2/tdm64-gcc-10.3.0-2.exe)
2. 安装 [Wails](https://wails.io/docs/gettingstarted/installation)

#### Linux 编译环境

```bash
# 安装 GCC 工具链
sudo apt-get install build-essential

# 安装 Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

#### macOS 编译环境

```bash
# 安装 Xcode Command Line Tools
xcode-select --install

# 安装 Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 编译步骤

```bash
# 克隆仓库
git clone https://github.com/a121400/sunnymcptool.git
cd sunnymcptool

# 安装前端依赖
cd frontend
npm install
cd ..

# 编译项目
wails build

# 开发模式运行
wails dev
```

## 使用说明

### 基础使用

1. 启动应用程序
2. 配置网络驱动（根据您的操作系统选择合适的驱动）
3. 启动抓包
4. 查看和分析网络请求
5. 可选：修改请求/响应数据

### 高级功能

- **证书管理**：安装和管理 HTTPS 抓包证书
- **JavaScript 脚本**：编写自定义脚本处理网络数据
- **进程过滤**：只抓取指定进程的网络流量
- **HOSTS 规则**：自定义域名解析规则

详细使用文档请参考 [SunnyNet 官方文档](https://github.com/qtgolang/SunnyNet)。

## 项目结构

```
sunnymcptool/
├── frontend/          # Vue 前端代码
│   ├── src/          # 源代码
│   │   ├── components/  # Vue 组件
│   │   └── main.js     # 入口文件
│   └── package.json   # 前端依赖
├── SunnyNet/         # SunnyNet 核心库（子模块）
├── CommAnd/          # 跨平台命令实现
├── MapHash/          # 哈希映射工具
├── mcp_standalone/   # MCP 独立服务器
├── *.go              # Go 后端代码
├── go.mod            # Go 模块配置
├── wails.json        # Wails 配置
└── README.md         # 项目文档
```

## 核心依赖

本项目核心功能基于以下开源项目：

- [SunnyNet](https://github.com/qtgolang/SunnyNet) - 网络中间件核心库
- [Wails](https://github.com/wailsapp/wails) - Go + Web 桌面应用框架
- [Vue 3](https://vuejs.org/) - 前端框架

## 贡献指南

欢迎贡献代码、报告问题或提出建议！

### 贡献流程

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m '添加某个很棒的功能'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

### 代码规范

- Go 代码遵循 `gofmt` 格式化标准
- Vue 代码遵循 Vue 3 官方风格指南
- 提交信息使用清晰的中文描述

## 技术支持与反馈

- **项目网站**: [https://esunny.vip/](https://esunny.vip/)
- **QQ 交流群**:
  - 一群：751406884
  - 二群：545120699
  - 三群：170902713
  - 四群：1070797457

## 注意事项

1. 如需支持 Windows 7 系统，请使用 Go 1.21 以下版本编译（例如 go 1.20.4）
2. Windows 编译请使用 [TDM-GCC](https://github.com/jmeubank/tdm-gcc/releases/download/v10.3.0-tdm64-2/tdm64-gcc-10.3.0-2.exe)
3. 本工具仅供学习和研究使用，请勿用于非法用途
4. 使用本工具抓取 HTTPS 流量需要安装证书

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 致谢

感谢 [SunnyNet](https://github.com/qtgolang/SunnyNet) 项目提供的强大网络中间件功能。

---

<div align="center">
如果这个项目对您有帮助，请给一个 ⭐ Star 支持一下！
</div>

