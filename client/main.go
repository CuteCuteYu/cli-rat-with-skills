package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	serverAddr   = "http://localhost:8080"
	pollInterval = 1 * time.Second
	clientID     = ""
	clientName   = "client"
	clientFile   = "client.id"
)

type RegisterRequest struct {
	Name string `json:"name"`
}

type RegisterResponse struct {
	ClientID string `json:"client_id"`
	Status   string `json:"status"`
}

type Command struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	ClientID string `json:"client_id"`
	Status   string `json:"status"`
}

type SubmitRequest struct {
	CommandID string `json:"command_id"`
	Status    string `json:"status"`
	Result    string `json:"result"`
}

func register() error {
	req := RegisterRequest{
		Name: clientName,
	}

	data, _ := json.Marshal(req)

	resp, err := http.Post(serverAddr+"/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("注册失败: %v", err)
	}
	defer resp.Body.Close()

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	clientID = result.ClientID
	saveClientID(clientID)
	fmt.Printf("[注册成功] ClientID: %s\n", clientID)
	return nil
}

func verifyClientID() bool {
	if clientID == "" {
		return false
	}

	resp, err := http.Get(fmt.Sprintf("%s/poll?client_id=%s", serverAddr, clientID))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	if result["status"] == "error" || result["status"] == "no_command" {
		return true
	}

	return false
}

func saveClientID(id string) {
	os.WriteFile(clientFile, []byte(id), 0644)
}

func loadClientID() string {
	data, err := os.ReadFile(clientFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func unregister() {
	if clientID == "" {
		return
	}

	req := map[string]string{"client_id": clientID}
	data, _ := json.Marshal(req)
	http.Post(serverAddr+"/unregister", "application/json", bytes.NewBuffer(data))
	os.Remove(clientFile)
}

func pollAndExecute() {
	for {
		if clientID == "" {
			time.Sleep(pollInterval)
			continue
		}

		resp, err := http.Get(fmt.Sprintf("%s/poll?client_id=%s", serverAddr, clientID))
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			time.Sleep(pollInterval)
			continue
		}
		resp.Body.Close()

		if result["status"] == "no_command" {
			time.Sleep(pollInterval)
			continue
		}

		cmdID, _ := result["id"]
		cmdContent, _ := result["content"]

		// 检查是否为退出命令
		if cmdContent == "__EXIT__" {
			fmt.Println("[服务端请求退出]")
			submitReq := SubmitRequest{
				CommandID: cmdID,
				Status:    "completed",
				Result:    "client exited",
			}
			submitData, _ := json.Marshal(submitReq)
			http.Post(serverAddr+"/result", "application/json", bytes.NewBuffer(submitData))
			unregister()
			os.Exit(0)
		}

		fmt.Printf("[执行命令] %s\n", cmdContent)

		output, err := executeCommand(cmdContent)
		status := "completed"
		if err != nil {
			status = "failed"
			output = err.Error()
		}

		submitReq := SubmitRequest{
			CommandID: cmdID,
			Status:    status,
			Result:    output,
		}

		submitData, _ := json.Marshal(submitReq)
		http.Post(serverAddr+"/result", "application/json", bytes.NewBuffer(submitData))

		fmt.Printf("[完成] 状态: %s\n", status)
		time.Sleep(pollInterval)
	}
}

func executeCommand(cmd string) (string, error) {
	if cmd == "" {
		return "", fmt.Errorf("空命令")
	}

	// Windows: 使用 cmd.exe /c 执行命令（支持内置命令如 dir, cd 等）
	cmdExec := exec.Command("cmd.exe", "/c", cmd)

	output, err := cmdExec.CombinedOutput()
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func main() {
	fmt.Println("=== 客户端启动 ===")
	fmt.Printf("服务端地址: %s\n", serverAddr)
	fmt.Printf("轮询间隔: %v\n", pollInterval)

	clientID = loadClientID()
	if clientID != "" {
		fmt.Printf("[恢复] 已有ClientID: %s\n", clientID)
		if !verifyClientID() {
			fmt.Println("[验证] ClientID无效，重新注册...")
			clientID = ""
			os.Remove(clientFile)
		}
	}

	if err := register(); err != nil {
		fmt.Printf("[注册失败] %v\n", err)
		fmt.Println("将在轮询时重试...")
	}

	go pollAndExecute()

	fmt.Println("[监听] 按回车键退出...")
	bufio.NewReader(os.Stdin).ReadLine()
	unregister()
	fmt.Println("客户端已退出")
}
