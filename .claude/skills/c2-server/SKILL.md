---
name: c2-server
description: 命令分发服务端控制工具。用于管理客户端、发送远程命令、查看执行历史。当用户需要启动/停止服务端、管理客户端连接、发送命令给客户端或查看命令历史时使用此技能。假设 server.exe 已在系统 PATH 环境变量中。
---

# C2 Server 控制工具

服务端命令分发系统的管理工具，用于管理客户端连接和远程命令执行。

## 快速参考

```bash
# 启动服务端（默认 0.0.0.0:8080）
server -init

# 自定义监听地址
server -init -ip 0.0.0.0 -port 9090

# 查看在线客户端
server list

# 发送命令
server send <client_id> <command>

# 断开客户端
server kill <client_id>

# 查看命令历史
server history
server show <command_id>

# 停止服务端
server stop

# 查看状态
server status
```

## 服务端管理

### 启动服务端

```bash
# 默认配置
server -init

# 自定义 IP 和端口
server -init -ip 192.168.1.100 -port 9999
```

启动后服务端在后台运行，输出 PID 信息。

### 停止服务端

```bash
server stop
```

会自动清理 `.lock` 和 `.pid` 文件。

### 查看状态

```bash
server status
```

显示服务端运行状态和 PID。

## 客户端管理

### 查看在线客户端

```bash
server list
```

输出格式：
```
[1] client-20260403013011 (127.0.0.1:54321) 最后活跃: 2026-04-03 14:30:00
[2] client-20260403014522 (192.168.1.50:54322) 最后活跃: 2026-04-03 14:29:55
```

### 断开客户端

```bash
server kill <client_id>
```

发送 `__EXIT__` 信号给客户端，使其主动退出并注销。

## 命令操作

### 发送命令

```bash
server send <client_id> <command>
```

**重要限制**：每次只能执行**单行命令**。如需完成多步骤任务，必须**多次发送单行命令**。

**示例：**

```bash
# 单行命令示例
server send client-xxx "dir"
server send client-xxx "ipconfig /all"

# PowerShell 单行命令
server send client-xxx "powershell Get-Process"
server send client-xxx "powershell Get-Service"

# 多步骤任务示例：需要分多次执行
# 步骤1：切换目录
server send client-xxx "cd C:\Temp"
# 步骤2：查看文件
server send client-xxx "dir"
# 步骤3：读取特定文件
server send client-xxx "type result.txt"

# 错误示例：不要尝试用 && 或 ; 连接多行命令
# server send client-xxx "cd C:\Temp && dir"  # 不支持
```

### 查看命令历史

```bash
# 查看最近 10 条命令列表
server history

# 查看具体命令详情和执行结果
server show <command_id>
```

**典型使用场景：**

```bash
# 场景1：检查命令是否执行成功
server history
# 输出示例：
# [1] cmd-20260403013032 -> dir 状态: completed
# [2] cmd-20260403014550 -> ipconfig 状态: failed

# 场景2：查看失败命令的错误信息
server show cmd-20260403014550
# 输出包含：命令内容、执行状态、错误结果

# 场景3：获取命令的执行结果
server show cmd-20260403013032
# 输出包含完整的 dir 命令结果

# 场景4：验证多步骤任务执行情况
# 执行完多个命令后查看历史，确认每个步骤的状态
server history
server show cmd-xxx  # 查看具体某步的结果
```

## 注意事项

1. **单行命令限制**：每次只能执行单行命令，不支持 `&&`、`||`、`;` 等多行连接符。多步骤任务必须拆分为多次单行命令执行
2. **命令执行**：客户端使用 `cmd.exe /c` 执行，支持 Windows 内置命令
3. **命令限制**：最多保存 10 条历史记录
4. **强制退出**：`kill` 命令会等待客户端确认退出（最多 5 秒）
5. **客户端管理**：客户端退出时需主动调用注销接口，否则需使用 `kill` 命令手动断开

## 文件说明

服务端运行时生成：
- `session.json` - 客户端会话
- `commands.json` - 命令历史
- `server.pid` - 进程 ID
- `server.lock` - 锁文件
- `server.log` - 运行日志
