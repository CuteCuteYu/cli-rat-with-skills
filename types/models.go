// Package types 定义了客户端和服务端之间共享的数据结构
package types

import "time"

// Client 客户端信息结构体
// 包含客户端的基本信息、注册时间和最后活跃时间
type Client struct {
	ID           string    `json:"id"`           // 客户端唯一标识符，格式为 client-时间戳
	Name         string    `json:"name"`         // 客户端名称
	Address      string    `json:"address"`      // 客户端网络地址 IP:Port
	RegisteredAt time.Time `json:"registered_at"` // 客户端注册时间
	LastPoll     time.Time `json:"last_poll"`     // 客户端最后一次轮询时间
}

// Command 命令信息结构体
// 表示服务端向客户端发送的命令及其执行状态
type Command struct {
	ID          string    `json:"id"`           // 命令唯一标识符，格式为 cmd-时间戳
	Content     string    `json:"content"`      // 要执行的命令内容
	ClientID    string    `json:"client_id"`    // 目标客户端ID
	Status      string    `json:"status"`       // 命令状态：pending(待执行)/executing(执行中)/completed(完成)/failed(失败)
	Result      string    `json:"result,omitempty"` // 命令执行结果（可选）
	CreatedAt   time.Time `json:"created_at"`   // 命令创建时间
	CompletedAt time.Time `json:"completed_at,omitempty"` // 命令完成时间（可选）
}

// Session 会话信息结构体
// 保存所有已注册的客户端信息
type Session struct {
	Clients []Client `json:"clients"` // 客户端列表
}

// CommandStore 命令存储结构体
// 用于持久化保存所有命令记录
type CommandStore struct {
	Commands []Command `json:"commands"` // 命令列表
}

// ===== API 请求/响应结构 =====

// RegisterRequest 客户端注册请求
type RegisterRequest struct {
	Name string `json:"name"` // 客户端名称
}

// RegisterResponse 客户端注册响应
type RegisterResponse struct {
	ClientID string `json:"client_id"` // 分配的客户端ID
	Status   string `json:"status"`    // 注册状态
}

// PollResponse 轮询命令响应
type PollResponse struct {
	ID       string `json:"id,omitempty"`       // 命令ID（有命令时返回）
	Content  string `json:"content,omitempty"`  // 命令内容（有命令时返回）
	ClientID string `json:"client_id,omitempty"` // 目标客户端ID（有命令时返回）
	Status   string `json:"status"`             // 响应状态：no_command(无命令)/有命令时返回命令详情
}

// SubmitRequest 提交命令结果请求
type SubmitRequest struct {
	CommandID string `json:"command_id"` // 命令ID
	Status    string `json:"status"`     // 执行状态：completed/failed
	Result    string `json:"result"`     // 执行结果
}

// SubmitResponse 提交命令结果响应
type SubmitResponse struct {
	Status string `json:"status"` // 提交状态
}

// UnregisterRequest 客户端注销请求
type UnregisterRequest struct {
	ClientID string `json:"client_id"` // 要注销的客户端ID
}

// UnregisterResponse 客户端注销响应
type UnregisterResponse struct {
	Status string `json:"status"` // 注销状态
}
