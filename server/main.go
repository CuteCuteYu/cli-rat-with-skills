package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	sessionFile = "session.json"
	maxCommands = 10
	mu          sync.RWMutex
	initFlag    bool
	serverFlag  bool
	pidFile     = "server.pid"
	logFile     = "server.log"
	lockFile    = "server.lock"
	listenIP    string
	listenPort  int
)

func init() {
	flag.BoolVar(&initFlag, "init", false, "启动服务端（后台运行）")
	flag.BoolVar(&serverFlag, "server", false, "服务端内部运行模式")
	flag.StringVar(&listenIP, "ip", "0.0.0.0", "监听IP地址")
	flag.IntVar(&listenPort, "port", 8080, "监听端口")
	loadSession()
}

func acquireLock() bool {
	if _, err := os.Stat(lockFile); err == nil {
		return false
	}
	f, err := os.Create(lockFile)
	if err != nil {
		return false
	}
	f.WriteString(fmt.Sprintf("%d", os.Getpid()))
	f.Close()
	return true
}

func releaseLock() {
	os.Remove(lockFile)
}

func savePID(pid int) {
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func loadPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	var pid int
	fmt.Sscanf(string(data), "%d", &pid)
	return pid, nil
}

func removePID() {
	os.Remove(pidFile)
}

func listenAddr() string {
	return fmt.Sprintf("%s:%d", listenIP, listenPort)
}

func detachProcess() error {
	execName, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	absPath, err := filepath.Abs(execName)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	logF, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	env := append(os.Environ(), "SERVER_MODE=daemon")
	procAttr := &os.ProcAttr{
		Files: []*os.File{logF, logF, logF},
		Env:   env,
	}

	process, err := os.StartProcess(absPath, []string{absPath, "-server", "-ip", listenIP, "-port", fmt.Sprintf("%d", listenPort)}, procAttr)
	logF.Close()

	if err != nil {
		return fmt.Errorf("启动进程失败: %v", err)
	}

	savePID(process.Pid)
	fmt.Println("=== 服务端已启动 ===")
	fmt.Printf("进程 PID: %d\n", process.Pid)
	return nil
}

type Client struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address"`
	RegisteredAt time.Time `json:"registered_at"`
	LastPoll     time.Time `json:"last_poll"`
}

type Session struct {
	Clients []Client `json:"clients"`
	mu      sync.RWMutex
}

var session Session

func init() {
	loadSession()
}

func loadSession() {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			session = Session{Clients: []Client{}}
			saveSession()
			return
		}
		fmt.Printf("加载会话文件失败: %v\n", err)
		return
	}

	if err := json.Unmarshal(data, &session); err != nil {
		fmt.Printf("解析会话文件失败: %v\n", err)
		session = Session{Clients: []Client{}}
	}
}

func saveSession() {
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		fmt.Printf("序列化会话失败: %v\n", err)
		return
	}

	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		fmt.Printf("保存会话失败: %v\n", err)
	}
}

func registerClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
		Port int    `json:"port,omitempty"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求失败", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	// 使用客户端真实连接地址
	displayAddr := r.RemoteAddr
	mu.Lock()
	for i, c := range session.Clients {
		if c.Name == req.Name {
			session.Clients = append(session.Clients[:i], session.Clients[i+1:]...)
			fmt.Printf("[移除] 重复客户端: %s (ID: %s)\n", req.Name, c.ID)
			break
		}
	}

	clientID := fmt.Sprintf("client-%s", time.Now().Format("20060102150405"))
	client := Client{
		ID:           clientID,
		Name:         req.Name,
		Address:      displayAddr,
		RegisteredAt: time.Now(),
		LastPoll:     time.Now(),
	}

	session.Clients = append(session.Clients, client)
	mu.Unlock()

	saveSession()

	resp := map[string]interface{}{
		"client_id": clientID,
		"status":    "registered",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	fmt.Printf("[注册] 客户端: %s (%s) 已注册, ID: %s\n", req.Name, client.Address, clientID)
}

func pollCommands(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "缺少client_id参数", http.StatusBadRequest)
		return
	}

	mu.Lock()
	for i := range session.Clients {
		if session.Clients[i].ID == clientID {
			session.Clients[i].LastPoll = time.Now()
			break
		}
	}
	mu.Unlock()
	saveSession()

	cmd := getPendingCommand(clientID)

	w.Header().Set("Content-Type", "application/json")
	if cmd == nil {
		json.NewEncoder(w).Encode(map[string]string{"status": "no_command"})
		return
	}

	cmd.Status = "executing"
	saveCommand(cmd)

	json.NewEncoder(w).Encode(cmd)
}

func submitResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	var result struct {
		CommandID string `json:"command_id"`
		Status    string `json:"status"`
		Result    string `json:"result"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求失败", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &result); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	cmd := loadCommand(result.CommandID)
	if cmd == nil {
		http.Error(w, "命令不存在", http.StatusNotFound)
		return
	}

	cmd.Status = result.Status
	cmd.Result = result.Result
	cmd.CompletedAt = time.Now()

	saveCommand(cmd)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	fmt.Printf("[结果] 命令 %s 执行完成, 状态: %s\n", cmd.ID, cmd.Status)
}

func unregisterClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ClientID string `json:"client_id"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求失败", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析JSON失败", http.StatusBadRequest)
		return
	}

	if removeClient(req.ClientID) {
		fmt.Printf("[注销] 客户端 %s 已注销\n", req.ClientID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "unregistered"})
		return
	}

	http.Error(w, "客户端不存在", http.StatusNotFound)
}

func sendCommand(clientID, cmdContent string) (*Command, error) {
	mu.RLock()
	found := false
	for i := range session.Clients {
		if session.Clients[i].ID == clientID {
			found = true
			break
		}
	}
	mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("客户端不存在: %s", clientID)
	}

	cmd := &Command{
		ID:        fmt.Sprintf("cmd-%s", time.Now().Format("20060102150405")),
		Content:   cmdContent,
		ClientID:  clientID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	saveCommand(cmd)

	fmt.Printf("[命令] 已发送给 %s: %s\n", clientID, cmdContent)

	return cmd, nil
}

func listClients() []Client {
	mu.RLock()
	defer mu.RUnlock()

	clients := make([]Client, len(session.Clients))
	copy(clients, session.Clients)
	return clients
}

func removeClient(clientID string) bool {
	mu.Lock()
	defer mu.Unlock()

	for i, c := range session.Clients {
		if c.ID == clientID {
			session.Clients = append(session.Clients[:i], session.Clients[i+1:]...)
			saveSession()
			return true
		}
	}
	return false
}

func getClientByID(id string) *Client {
	mu.RLock()
	defer mu.RUnlock()

	for _, c := range session.Clients {
		if c.ID == id {
			return &c
		}
	}
	return nil
}

func main() {
	flag.Parse()

	if initFlag {
		if !acquireLock() {
			fmt.Println("错误: 服务端已在运行")
			fmt.Println("使用 server.exe status 查看状态")
			return
		}

		if err := detachProcess(); err != nil {
			fmt.Printf("启动失败: %v\n", err)
			releaseLock()
			return
		}
		return
	}

	if os.Getenv("SERVER_MODE") == "daemon" {
		logF, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			defer logF.Close()
			fmt.Fprintln(logF, "[服务端启动]")
		}

		http.HandleFunc("/register", registerClient)
		http.HandleFunc("/unregister", unregisterClient)
		http.HandleFunc("/poll", pollCommands)
		http.HandleFunc("/result", submitResult)

		if logF != nil {
			fmt.Fprintf(logF, "[HTTP服务] 启动成功 http://%s\n", listenAddr())
		}
		fmt.Printf("[HTTP服务] 启动成功 http://%s\n", listenAddr())

		if err := http.ListenAndServe(listenAddr(), nil); err != nil {
			if logF != nil {
				fmt.Fprintf(logF, "HTTP服务启动失败: %v\n", err)
			}
		}
		return
	}

	if initFlag {
		if err := detachProcess(); err != nil {
			fmt.Printf("启动失败: %v\n", err)
		}
		return
	}

	if flag.Arg(0) == "-server" {
		logF, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			defer logF.Close()
			fmt.Fprintln(logF, "[服务端启动]")
		}

		http.HandleFunc("/register", registerClient)
		http.HandleFunc("/unregister", unregisterClient)
		http.HandleFunc("/poll", pollCommands)
		http.HandleFunc("/result", submitResult)

		if logF != nil {
			fmt.Fprintf(logF, "[HTTP服务] 启动成功 http://%s\n", listenAddr())
		}
		fmt.Printf("[HTTP服务] 启动成功 http://%s\n", listenAddr())

		if err := http.ListenAndServe(listenAddr(), nil); err != nil {
			if logF != nil {
				fmt.Fprintf(logF, "HTTP服务启动失败: %v\n", err)
			}
		}
		return
	}

	if initFlag {
		if err := detachProcess(); err != nil {
			fmt.Printf("启动失败: %v\n", err)
		}
		return
	}

	if flag.Arg(0) == "stop" {
		pid, err := loadPID()
		if err != nil {
			fmt.Println("服务端未运行")
			return
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println("找不到服务端进程")
			removePID()
			return
		}

		proc.Kill()
		removePID()
		releaseLock()
		fmt.Println("=== 服务端已停止 ===")
		return
	}

	if flag.Arg(0) == "-server" {
		http.HandleFunc("/register", registerClient)
		http.HandleFunc("/poll", pollCommands)
		http.HandleFunc("/result", submitResult)

		go func() {
			fmt.Printf("[HTTP服务] 启动成功 http://%s\n", listenAddr())
			if err := http.ListenAndServe(listenAddr(), nil); err != nil {
				fmt.Printf("HTTP服务启动失败: %v\n", err)
			}
		}()

		fmt.Println("=== 服务端运行中 ===")
		fmt.Println("按 Ctrl+C 停止")
		select {}
	}

	if flag.NArg() == 0 {
		showHelp()
		return
	}

	cmd := flag.Arg(0)
	switch cmd {
	case "list":
		listClientsCmd()
	case "send":
		if flag.NArg() < 3 {
			fmt.Println("用法: server.exe send <client_id> <command>")
			return
		}
		clientID := flag.Arg(1)
		cmdContent := strings.Join(flag.Args()[2:], " ")
		_, err := sendCommand(clientID, cmdContent)
		if err != nil {
			fmt.Printf("发送失败: %v\n", err)
		} else {
			fmt.Println("命令已发送")
		}
	case "history":
		historyCmd()
	case "show":
		if flag.NArg() < 2 {
			fmt.Println("用法: server.exe show <command_id>")
			return
		}
		cmdID := flag.Arg(1)
		showCommandCmd(cmdID)
	case "kill":
		if flag.NArg() < 2 {
			fmt.Println("用法: server.exe kill <client_id>")
			return
		}
		clientID := flag.Arg(1)
		killClientCmd(clientID)
	case "help", "-h", "--help":
		showHelp()
	case "stop":
		fmt.Println("请使用 server.exe stop 命令停止服务端")
	case "status":
		checkStatus()
	default:
		fmt.Printf("未知命令: %s\n", cmd)
		fmt.Println("输入 server.exe help 查看可用命令")
	}
}

func showHelp() {
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

func checkStatus() {
	pid, err := loadPID()
	if err != nil {
		fmt.Println("状态: 未运行")
		fmt.Println("服务端未启动或已停止")
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil || proc == nil {
		fmt.Println("状态: 未运行")
		fmt.Println("PID文件存在但进程已退出")
		removePID()
		return
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/register", listenPort))
	if err != nil {
		fmt.Println("状态: 运行中")
		fmt.Printf("PID: %d\n", pid)
		fmt.Println("HTTP服务可能未响应")
		return
	}
	resp.Body.Close()

	fmt.Println("状态: 运行中")
	fmt.Printf("PID: %d\n", pid)
}

func listClientsCmd() {
	clients := listClients()
	if len(clients) == 0 {
		fmt.Println("暂无在线客户端")
	} else {
		for i, c := range clients {
			fmt.Printf("[%d] %s (%s) 最后活跃: %s\n",
				i+1, c.ID, c.Address, c.LastPoll.Format("2006-01-02 15:04:05"))
		}
	}
}

func historyCmd() {
	commands := listCommands()
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

func showCommandCmd(cmdID string) {
	cmd := getCommandByID(cmdID)
	if cmd == nil {
		fmt.Println("命令不存在")
		return
	}
	fmt.Printf("ID: %s\n", cmd.ID)
	fmt.Printf("内容: %s\n", cmd.Content)
	fmt.Printf("客户端: %s\n", cmd.ClientID)
	fmt.Printf("状态: %s\n", cmd.Status)
	fmt.Printf("创建时间: %s\n", cmd.CreatedAt.Format("2006-01-02 15:04:05"))
	if cmd.CompletedAt.Unix() > 0 {
		fmt.Printf("完成时间: %s\n", cmd.CompletedAt.Format("2006-01-02 15:04:05"))
	}
	if cmd.Result != "" {
		fmt.Printf("结果:\n%s\n", cmd.Result)
	}
}

func killClientCmd(clientID string) {
	client := getClientByID(clientID)
	if client == nil {
		fmt.Printf("错误: 客户端不存在: %s\n", clientID)
		return
	}

	// 发送退出命令给客户端
	cmd, err := sendCommand(clientID, "__EXIT__")
	if err != nil {
		fmt.Printf("错误: 发送退出命令失败: %v\n", err)
		return
	}

	// 等待客户端执行退出命令
	fmt.Printf("[等待] 等待客户端退出...\n")
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		updatedCmd := loadCommand(cmd.ID)
		if updatedCmd != nil && (updatedCmd.Status == "completed" || updatedCmd.Status == "failed") {
			break
		}
	}

	// 从列表中移除客户端
	if removeClient(clientID) {
		fmt.Printf("[断开] 客户端已断开连接: %s (%s)\n", clientID, client.Name)
	} else {
		fmt.Printf("错误: 无法移除客户端: %s\n", clientID)
	}
}
