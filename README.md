# 命令分发系统

一个基于 Go 语言开发的客户端-服务器架构命令分发系统。服务端可以管理多个客户端，向指定客户端发送系统命令并获取执行结果。

## 功能特性

- **多客户端管理**：支持同时管理多个客户端连接
- **命令分发**：向指定客户端发送命令并远程执行
- **结果回传**：客户端执行命令后将结果返回给服务端
- **历史记录**：保存命令执行历史，可查询命令详情
- **后台运行**：服务端支持以守护进程方式运行
- **强制断开**：服务端可主动断开指定客户端连接

## 系统架构

```
┌─────────────┐                          ┌─────────────┐
│   Server    │                          │   Client    │
│  :8080      │                          │             │
├─────────────┤                          ├─────────────┤
│             │  HTTP JSON               │             │
│  HTTP Server│◄─────────────────────────┤  Poll Loop  │
│             │       1秒/次              │             │
│             │                          ├─────────────┤
│ Command Q   │                          │   cmd.exe   │
│ Session Mgr │─── send command ────────►│   /c        │
│             │                          │             │
└─────────────┘◄── return result ───────└─────────────┘
```

**工作流程：**
1. 客户端启动后向服务端注册，获取唯一 ClientID
2. 客户端每秒轮询一次服务端，获取待执行命令
3. 服务端通过 `send` 命令将命令加入队列
4. 客户端获取命令后通过 `cmd.exe /c` 执行
5. 执行结果通过 `/result` 接口返回给服务端
6. 服务端保存命令历史和执行结果

## 环境要求

- Go 1.26+
- Windows 操作系统
- 局域网环境或本地运行

## 安装与编译

### 前置条件

1. **安装 Go 语言环境**

   下载并安装 Go 1.26 或更高版本：
   - 访问 [https://golang.org/dl/](https://golang.org/dl/)
   - 下载 Windows 安装包并安装
   - 验证安装：
     ```powershell
     go version
     ```

2. **配置环境变量（可选）**

   如果需要自定义 GOPATH：
   ```powershell
   # 设置 GOPATH
   $env:GOPATH = "C:\Go\Projects"

   # 添加到 PATH
   $env:PATH += ";$env:GOPATH\bin"
   ```

### 编译步骤

#### 方式一：使用构建脚本（推荐）

```powershell
# 进入项目目录
cd C:\Users\j4543\Desktop\test\NEWTEST

# 执行构建脚本
powershell -File build.ps1
```

构建脚本会自动编译服务端和客户端，并显示编译结果：
```
[OK] Server built successfully
[OK] Client built successfully

Build complete!
```

#### 方式二：手动编译

```powershell
# 进入项目目录
cd C:\Users\j4543\Desktop\test\NEWTEST

# 编译服务端（包含 main.go 和 command.go）
go build -o server.exe server/main.go server/command.go

# 编译客户端
go build -o client.exe client/main.go
```

#### 方式三：交叉编译

如需为其他平台编译：

```powershell
# Linux 64位
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o server-linux server/main.go server/command.go
go build -o client-linux client/main.go

# macOS 64位
$env:GOOS = "darwin"
$env:GOARCH = "amd64"
go build -o server-mac server/main.go server/command.go
go build -o client-mac client/main.go

# 恢复 Windows 编译
$env:GOOS = "windows"
$env:GOARCH = "amd64"
```

#### 编译优化选项

```powershell
# 减小可执行文件体积
go build -ldflags "-s -w" -o server.exe server/main.go server/command.go
go build -ldflags "-s -w" -o client.exe client/main.go

# 完整优化（推荐发布时使用）
go build -ldflags "-s -w" -trimpath -o server.exe server/main.go server/command.go
go build -ldflags "-s -w" -trimpath -o client.exe client/main.go
```

### 编译产物

编译完成后生成以下文件：

| 文件 | 大小（约） | 说明 |
|------|-----------|------|
| `server.exe` | ~2 MB | 服务端程序 |
| `client.exe` | ~2 MB | 客户端程序 |

### 验证编译结果

```powershell
# 查看服务端帮助
server help

# 查看客户端版本（启动即显示）
.\client.exe
```

## 使用说明

### 环境配置（推荐）

将 `server.exe` 添加到系统 PATH 环境变量后，可在任意目录直接使用 `server` 命令。

#### Windows 添加到 PATH

**方法一：临时添加（当前会话）**

```powershell
# 添加到当前会话 PATH
$env:PATH += ";C:\Users\j4543\Desktop\test\NEWTEST"

# 验证
server help
```

**方法二：永久添加（PowerShell）**

```powershell
# 添加到用户 PATH（永久）
[Environment]::SetEnvironmentVariable("Path", $env:PATH + ";C:\Users\j4543\Desktop\test\NEWTEST", "User")

# 刷新环境变量
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","User")
```

**方法三：图形界面添加**

1. 右键「此电脑」→「属性」
2. 「高级系统设置」→「环境变量」
3. 在「用户变量」中找到 `Path`，点击「编辑」
4. 点击「新建」，添加 `C:\Users\j4543\Desktop\test\NEWTEST`
5. 确定保存，重新打开终端

#### Claude Code Skill 使用

本项目包含 `c2-server` skill，让您可以在 Claude Code 中使用自然语言操作服务端。

##### Skill 文件位置

Skill 文件位于项目根目录：
```
.claude/skills/c2-server/SKILL.md
```

##### 安装 Skill

**方式一：复制到 Claude Code skills 目录（推荐）**

```powershell
# 创建 Claude Code skills 目录（如果不存在）
mkdir -p ~/.claude/skills/c2-server

# 复制 skill 文件
copy .claude\skills\c2-server\SKILL.md ~/.claude/skills/c2-server/SKILL.md
```

**方式二：符号链接（便于开发更新）**

```powershell
# 在 Windows 上使用 mklink
mklink /D "C:\Users\YourUsername\.claude\skills\c2-server" "D:\code\go\cli-rat\cli-rat-with-skills\.claude\skills\c2-server"
```

##### 使用 Skill

安装后，您可以在 Claude Code 中使用自然语言操作服务端：

```
你：启动服务端
Claude：server -init

你：查看在线客户端
Claude：server list

你：向 client-xxx 发送 ipconfig 命令
Claude：server send client-xxx "ipconfig"

你：查看命令历史
Claude：server history

你：断开客户端 client-xxx
Claude：server kill client-xxx
```

##### Skill 支持的操作

- 启动/停止服务端
- 查看在线客户端列表
- 发送命令到指定客户端
- 查看命令执行历史
- 查看命令详情和结果
- 断开客户端连接

##### Skill 文件内容

完整的 Skill 文件位于 `.claude/skills/c2-server/SKILL.md`，包含所有命令的详细说明和示例。

### 服务端命令

#### 启动/停止服务端

```powershell
# 已添加 PATH 后（推荐）
server -init

# 或使用完整路径（未添加 PATH 时）
.\server.exe -init

# 自定义监听地址和端口
server -init -ip 0.0.0.0 -port 9090

# 停止服务端
server stop

# 检查服务端状态
server status
```

#### 客户端管理

```powershell
# 查看在线客户端列表
server list

# 断开指定客户端连接
server kill <client_id>
```

#### 命令操作

```powershell
# 向客户端发送命令
server send <client_id> <command>

# 示例：查看客户端目录
server send client-20260403013011 "dir"

# 示例：执行 PowerShell 命令
server send client-20260403013011 "powershell Get-Process"

# 示例：查看系统信息
server send client-20260403013011 "ipconfig /all"
```

#### 历史记录

```powershell
# 查看命令历史（最近10条）
server history

# 查看指定命令详情
server show <command_id>

# 示例
server show cmd-20260403013032
```

#### 帮助信息

```powershell
server help
```

### 客户端使用

#### 修改服务端地址

编辑 `client/main.go` 中的 `serverAddr` 变量：

```go
var (
    serverAddr = "http://192.168.1.100:8080"  // 修改为服务端地址
    // ...
)
```

#### 启动客户端

```powershell
.\client.exe
```

启动后客户端会：
1. 向服务端注册并获取 ClientID
2. ClientID 保存到 `client.id` 文件，重启后自动恢复
3. 每秒轮询一次服务端获取命令
4. 收到命令后执行并返回结果
5. 按回车键退出

## 配置文件

服务端运行时会在当前目录生成以下文件：

| 文件 | 说明 |
|------|------|
| `session.json` | 客户端会话信息 |
| `commands.json` | 命令历史记录 |
| `server.pid` | 服务端进程 ID |
| `server.lock` | 服务端锁文件 |
| `server.log` | 服务端运行日志 |

客户端运行时生成：

| 文件 | 说明 |
|------|------|
| `client.id` | 客户端注册 ID |

## API 接口

### POST /register

客户端注册

**请求：**
```json
{
  "name": "client"
}
```

**响应：**
```json
{
  "client_id": "client-20260403013011",
  "status": "registered"
}
```

### GET /poll?client_id=xxx

客户端轮询命令

**响应（无命令）：**
```json
{
  "status": "no_command"
}
```

**响应（有命令）：**
```json
{
  "id": "cmd-20260403013032",
  "content": "dir",
  "client_id": "client-20260403013011",
  "status": "pending"
}
```

### POST /result

提交命令执行结果

**请求：**
```json
{
  "command_id": "cmd-20260403013032",
  "status": "completed",
  "result": "执行结果..."
}
```

**响应：**
```json
{
  "status": "success"
}
```

### POST /unregister

客户端注销

**请求：**
```json
{
  "client_id": "client-20260403013011"
}
```

## 注意事项

1. **命令执行**：客户端使用 `cmd.exe /c` 执行命令，支持 Windows 内置命令（如 `dir`、`cd`）和可执行文件
2. **强制退出**：使用 `kill` 命令会向客户端发送 `__EXIT__` 信号，客户端收到后会主动退出
3. **命令历史**：最多保存最近 10 条命令记录
4. **网络要求**：客户端需要能访问服务端的 HTTP 端口
5. **客户端管理**：客户端退出时需主动调用注销接口，否则需使用 `kill` 命令手动断开

## 安全提示

本系统设计用于局域网内的受控环境。在生产环境中使用时应考虑：
- 添加身份认证机制
- 使用 HTTPS 加密通信
- 限制命令执行权限
- 添加命令白名单/黑名单

## 项目结构

```
NEWTEST/
├── server/
│   ├── main.go      # 服务端主程序
│   └── command.go   # 命令管理模块
├── client/
│   └── main.go      # 客户端程序
├── build.ps1        # 构建脚本
├── go.mod           # Go 模块配置
└── README.md        # 本文件
```

## 许可证

本项目仅供学习和研究使用。
