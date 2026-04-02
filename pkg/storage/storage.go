// Package storage 提供文件存储操作功能
// 负责会话信息、命令记录和客户端ID的持久化存储
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"cli-rat/types"
)

const (
	// SessionFile 会话信息存储文件名
	SessionFile = "session.json"
	// CommandsFile 命令历史存储文件名
	CommandsFile = "commands.json"
	// MaxCommands 最大保存命令数量
	MaxCommands = 10
)

var (
	// mu 读写锁，保护并发访问
	mu sync.RWMutex
)

// ===== Session 管理 =====

// LoadSession 从文件加载会话信息
// 如果文件不存在或解析失败，返回空的会话
func LoadSession() (*types.Session, error) {
	data, err := os.ReadFile(SessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，返回空会话
			return &types.Session{Clients: []types.Client{}}, nil
		}
		return nil, fmt.Errorf("读取会话文件失败: %v", err)
	}

	var session types.Session
	if err := json.Unmarshal(data, &session); err != nil {
		// 解析失败，返回空会话
		return &types.Session{Clients: []types.Client{}}, nil
	}
	return &session, nil
}

// SaveSession 保存会话信息到文件
// 使用 JSON 格式，带缩进便于阅读
func SaveSession(session *types.Session) error {
	if session == nil {
		return fmt.Errorf("会话对象不能为空")
	}

	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化会话失败: %v", err)
	}

	if err := os.WriteFile(SessionFile, data, 0644); err != nil {
		return fmt.Errorf("保存会话失败: %v", err)
	}
	return nil
}

// ===== Command 管理 =====

// LoadCommands 从文件加载命令历史
// 如果文件不存在或解析失败，返回空的命令存储
func LoadCommands() (*types.CommandStore, error) {
	data, err := os.ReadFile(CommandsFile)
	if err != nil {
		return &types.CommandStore{Commands: []types.Command{}}, nil
	}

	var store types.CommandStore
	if err := json.Unmarshal(data, &store); err != nil {
		return &types.CommandStore{Commands: []types.Command{}}, nil
	}
	return &store, nil
}

// SaveCommands 保存命令存储到文件
func SaveCommands(store *types.CommandStore) error {
	if store == nil {
		return fmt.Errorf("命令存储对象不能为空")
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化命令失败: %v", err)
	}

	if err := os.WriteFile(CommandsFile, data, 0644); err != nil {
		return fmt.Errorf("保存命令失败: %v", err)
	}
	return nil
}

// GetPendingCommand 获取指定客户端的待执行命令
// 返回状态为 "pending" 的命令，如果没有则返回 nil
func GetPendingCommand(clientID string) *types.Command {
	if clientID == "" {
		return nil
	}

	store, _ := LoadCommands()
	if store == nil {
		return nil
	}

	for i := range store.Commands {
		if store.Commands[i].ClientID == clientID && store.Commands[i].Status == "pending" {
			return &store.Commands[i]
		}
	}
	return nil
}

// SaveCommand 保存或更新命令
// 如果命令已存在则更新，否则添加新命令
// 自动维护最大命令数量限制
func SaveCommand(cmd *types.Command) error {
	if cmd == nil {
		return fmt.Errorf("命令对象不能为空")
	}

	if cmd.ID == "" {
		return fmt.Errorf("命令ID不能为空")
	}

	store, err := LoadCommands()
	if err != nil {
		return err
	}

	// 查找并更新现有命令
	for i := range store.Commands {
		if store.Commands[i].ID == cmd.ID {
			store.Commands[i] = *cmd
			return SaveCommands(store)
		}
	}

	// 添加新命令
	store.Commands = append(store.Commands, *cmd)
	// 保持最大命令数量限制（FIFO）
	if len(store.Commands) > MaxCommands {
		store.Commands = store.Commands[len(store.Commands)-MaxCommands:]
	}
	return SaveCommands(store)
}

// GetCommandByID 根据命令ID获取命令
// 返回命令指针，如果不存在返回nil
func GetCommandByID(cmdID string) *types.Command {
	if cmdID == "" {
		return nil
	}

	store, _ := LoadCommands()
	if store == nil {
		return nil
	}

	for _, cmd := range store.Commands {
		if cmd.ID == cmdID {
			return &cmd
		}
	}
	return nil
}

// ListCommands 获取命令历史列表
// 返回最近的命令，最多 MaxCommands 条
func ListCommands() []types.Command {
	store, _ := LoadCommands()
	if store == nil {
		return []types.Command{}
	}

	commands := store.Commands
	if len(commands) > MaxCommands {
		commands = commands[len(commands)-MaxCommands:]
	}
	return commands
}

// ===== Client ID 管理 =====

// LoadClientID 从文件加载客户端ID
// 返回客户端ID，如果文件不存在或读取失败返回空字符串
func LoadClientID(filename string) string {
	if filename == "" {
		return ""
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}

	// 去除首尾空白字符
	return strings.TrimSpace(string(data))
}

// SaveClientID 保存客户端ID到文件
func SaveClientID(filename, id string) error {
	if filename == "" {
		return fmt.Errorf("文件名不能为空")
	}

	if id == "" {
		return fmt.Errorf("客户端ID不能为空")
	}

	return os.WriteFile(filename, []byte(id), 0644)
}
