# Android 云函数抓包集成到 Sunny 计划

## 目标

在 Sunny GUI 中新增一个按钮，点击后自动连接 Android 手机、启动 frida-server、注入云函数 Hook 脚本，将拦截到的数据实时显示在 Sunny 的抓包列表中。

---

## 架构方案

**纯 Go 实现**：通过 `os/exec` 调用 ADB 和 frida CLI（或使用 frida REST API），不依赖 Node.js。

```
[Sunny Go Backend]
    │
    ├── adb_frida.go         # ADB 管理 + frida-server 管理
    ├── cloud_hook.go        # 云函数 Hook 核心逻辑（frida 注入 + 消息接收）
    │
    ├── 通过 frida CLI 注入 JS 脚本到微信进程
    ├── 通过 frida 的 STDIO 管道接收 JSON 消息
    ├── 将消息转换为 ListInfo + Request 格式
    └── 调用 AddInsertList() 推入前端列表
```

---

## 数据流

```
Android 微信进程
    ↓ (Frida hook 拦截 operateWXData)
Frida Script (send JSON)
    ↓ (frida CLI stdout)
Go: cloud_hook.go (解析 JSON)
    ↓ (转换为 ListInfo)
Go: AddInsertList(&ListInfo{...})
    ↓ (100ms 定时批量推送)
Frontend: 抓包列表显示
```

---

## 核心文件

### 1. `adb_frida.go` - ADB 和 Frida 管理

```go
package main

// 功能：
// - CheckADBDevice() 检测 ADB 连接状态
// - GetDeviceArch() 获取设备 CPU 架构
// - IsFridaServerRunning() 检测 frida-server 是否以 root 运行
// - StartFridaServer() 启动 frida-server (su -c, setenforce 0)
// - InstallFridaServer() 下载并安装匹配版本的 frida-server
// - FindWeChatProcesses() 列出微信进程 (通过 adb shell ps)

const FRIDA_VERSION = "16.6.6" // 与 frida CLI 版本匹配
const FRIDA_SERVER_PATH = "/data/local/tmp/frida-server"
```

**关键实现细节**：
- 启动命令：`adb shell "su -c /data/local/tmp/frida-server -D"` (保持 root 上下文)
- 检测运行：`adb shell su -c ps -A` → 匹配 `root.*S\s+frida-server`
- 设备架构：`adb shell getprop ro.product.cpu.abi`
- SELinux：`adb shell su -c setenforce 0`

### 2. `cloud_hook.go` - 云函数 Hook 核心

```go
package main

// 功能：
// - StartCloudHook() 启动安卓云函数抓包
// - StopCloudHook() 停止
// - hookProcess(pid int) 注入脚本到单个进程
// - parseHookMessage(line string) 解析 frida 输出的 JSON

// Frida 脚本嵌入：将 cloudHookAndroid.js 作为 Go embed 资源
//go:embed frida-scripts/cloudHookAndroid.js
var cloudHookScript string
```

**注入方式**（两种可选）：

#### 方案 A：frida CLI（推荐，无需 Go binding）
```go
cmd := exec.Command("frida", "-U", "-p", strconv.Itoa(pid), "-l", scriptPath, "--no-pause")
stdout, _ := cmd.StdoutPipe()
// 逐行读取 stdout，每行是一条 JSON 消息
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    line := scanner.Text()
    // 解析并推入列表
}
```

#### 方案 B：frida REST API + WebSocket
```
frida-server 启动时加 --listen=127.0.0.1:27042
通过 adb forward tcp:27042 tcp:27042
然后 Go 通过 HTTP/WS 连接 frida REST API 注入脚本
```

### 3. 数据格式转换

Frida Hook 输出的 JSON：
```json
{
  "type": "cloud-request",
  "apiType": "operateWXData",
  "param1": "...",
  "param2": "{\"env\":\"...\",\"name\":\"getUser\",\"data\":{}}",
  "timestamp": 1782532062560
}
```

转换为 Sunny 的 `ListInfo`：
```go
func cloudMsgToListInfo(msg CloudHookMessage) *ListInfo {
    theology := HashMap.CreateUniqueID()
    
    // 存入 HashMap
    req := &MapHash.Request{
        PID:    "WeChat:Android",
        Method: "CLOUD",
        URL:    "wx-cloud://" + msg.ApiType + "/" + extractFuncName(msg.Param2),
        Body:   []byte(msg.Param2),
    }
    req.Display = true
    HashMap.SetRequest(theology, req)
    
    // 构造 ListInfo
    return &ListInfo{
        Theology:  theology,
        State:     "完成",
        URL:       req.URL,
        HOST:      "wx-cloud",
        Method:    "CLOUD",
        Ico:       "api",
        Type:      "application/json",
        SendTime:  time.Now().Format("15:04:05"),
        ClientIP:  "Android",
        PID:       "WeChat",
    }
}
```

对于 Response：
```go
func cloudResponseToUpdate(msg CloudHookMessage, theology int) {
    req := HashMap.GetRequest(theology)
    if req != nil {
        req.Response.Body = []byte(msg.RespJson)
        req.Response.StateCode = 200
        req.Response.Header = http.Header{"Content-Type": {"application/json"}}
    }
    // 通知前端更新
    UpdateData = append(UpdateData, &UpdateCurrentResponse{
        Theology:  theology,
        StateCode: 200,
        Body:      []byte(msg.RespJson),
    })
}
```

---

## 前端修改

### 按钮位置
在 Sunny 前端工具栏（顶部或侧边栏）添加一个 "安卓抓包" 按钮。

### 事件通信
```javascript
// 前端发送
window.go.main.App.Do({
    Command: "启动安卓云函数Hook",
    Args: {}
})

// 前端接收（自动通过现有的 "插入列表" 事件）
// 无需额外处理，数据格式与 HTTP 抓包一致
```

### Go 后端 event handler
在 `public.go` 的 `event` 函数中添加：
```go
case "启动安卓云函数Hook":
    go StartCloudHook()
    return map[string]interface{}{"success": true}
case "停止安卓云函数Hook":
    StopCloudHook()
    return map[string]interface{}{"success": true}
case "获取安卓Hook状态":
    return GetCloudHookStatus()
```

---

## 依赖

1. **frida CLI**：需要在 PATH 中有 `frida` 命令（`pip install frida-tools`）
   - 或者打包 `frida.exe` 到 Sunny 的 resources 目录
2. **ADB**：PATH 中需有 `adb` 命令
3. **frida-server**：自动下载安装到设备（与当前微信调试助手逻辑相同）

---

## 实现步骤

1. [ ] 创建 `adb_frida.go` - ADB/frida-server 管理
2. [ ] 创建 `cloud_hook.go` - Hook 核心逻辑
3. [ ] 嵌入 `cloudHookAndroid.js` 脚本（从微信调试助手项目复制）
4. [ ] 在 `public.go` 添加 event handler
5. [ ] 实现数据格式转换（CloudHookMessage → ListInfo/Request）
6. [ ] 前端添加 "安卓抓包" 按钮
7. [ ] 测试完整流程：按钮 → 连接设备 → 注入 → 数据显示

---

## 注意事项

- **Frida 版本匹配**：frida CLI 版本必须与设备上 frida-server 版本一致
- **多进程 Hook**：微信有多个进程（appbrand、主进程等），需要 hook appbrand 进程
- **数据区分**：云函数数据在列表中用特殊图标 `api` 和 Method `CLOUD` 区分
- **SELinux**：自动 setenforce 0
- **Root 权限**：frida-server 必须以 root 身份运行（通过 `su -c` 确保）
- **连接断开处理**：设备断开时自动停止 hook，清理 frida 进程

---

## 备注

- PC 微信抓包：Sunny 本身的代理功能已覆盖，无需额外实现
- 搜索/过滤：Sunny 列表自带筛选功能，云函数数据用 Method=`CLOUD` 标识即可被筛选
- 导出：Sunny 已有保存抓包数据功能，云函数数据作为普通记录一并保存
- wx.login 捕获：如有需要可后续单独加，当前只做 operateWXData 拦截
