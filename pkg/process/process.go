// Package process 提供进程管理功能
// 负责服务端的后台运行、PID管理和锁文件机制
package process

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// PIDFile 进程ID文件名
	PIDFile = "server.pid"
	// LockFile 锁文件名
	LockFile = "server.lock"
	// LogFile 日志文件名
	LogFile = "server.log"
)

// ===== 锁文件管理 =====
// 锁文件用于防止同时启动多个服务端实例

// AcquireLock 获取锁
// 如果锁文件已存在，返回 false（表示已被锁定）
// 否则创建锁文件并返回 true
func AcquireLock() bool {
	if _, err := os.Stat(LockFile); err == nil {
		// 锁文件已存在
		return false
	}
	// 创建锁文件
	f, err := os.Create(LockFile)
	if err != nil {
		return false
	}
	// 写入当前进程ID
	if _, err := f.WriteString(fmt.Sprintf("%d", os.Getpid())); err != nil {
		f.Close()
		os.Remove(LockFile)
		return false
	}
	f.Close()
	return true
}

// ReleaseLock 释放锁
// 删除锁文件，允许其他进程启动
func ReleaseLock() error {
	return os.Remove(LockFile)
}

// IsLocked 检查是否已被锁定
func IsLocked() bool {
	_, err := os.Stat(LockFile)
	return err == nil
}

// ===== PID 管理 =====
// PID 文件用于追踪后台运行的服务端进程

// SavePID 保存进程ID到文件
func SavePID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效的PID: %d", pid)
	}
	return os.WriteFile(PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// LoadPID 从文件加载进程ID
func LoadPID() (int, error) {
	data, err := os.ReadFile(PIDFile)
	if err != nil {
		return 0, fmt.Errorf("读取PID文件失败: %v", err)
	}

	var pid int
	n, err := fmt.Sscanf(string(data), "%d", &pid)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("解析PID失败: %v", err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("无效的PID值: %d", pid)
	}

	return pid, nil
}

// RemovePID 删除PID文件
func RemovePID() error {
	return os.Remove(PIDFile)
}

// ===== 进程操作 =====

// FindProcess 查找指定PID的进程
func FindProcess(pid int) (*os.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("无效的PID: %d", pid)
	}
	return os.FindProcess(pid)
}

// KillProcess 杀死指定PID的进程
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效的PID: %d", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("查找进程失败: %v", err)
	}

	if err := proc.Kill(); err != nil {
		return fmt.Errorf("杀死进程失败: %v", err)
	}

	return nil
}

// ===== 后台运行 =====

// DetachProcess 将当前进程分离为后台守护进程
// 启动一个新的进程实例，重定向输出到日志文件
// 返回新进程的句柄
func DetachProcess(listenIP string, listenPort int) (*os.Process, error) {
	// 验证参数
	if listenIP == "" {
		return nil, fmt.Errorf("监听IP不能为空")
	}
	if listenPort <= 0 || listenPort > 65535 {
		return nil, fmt.Errorf("端口号必须在 1-65535 之间，当前值: %d", listenPort)
	}

	// 获取当前可执行文件路径
	execName, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(execName)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %v", err)
	}

	// 创建/清空日志文件
	logF, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %v", err)
	}

	// 设置环境变量，标识为守护进程模式
	env := append(os.Environ(), "SERVER_MODE=daemon")

	// 配置新进程属性
	// 将标准输入、输出、错误都重定向到日志文件
	procAttr := &os.ProcAttr{
		Files: []*os.File{logF, logF, logF}, // stdin, stdout, stderr
		Env:   env,
	}

	// 构建启动参数
	args := []string{absPath, "-server", "-ip", listenIP, "-port", fmt.Sprintf("%d", listenPort)}

	// 启动新进程
	process, err := os.StartProcess(absPath, args, procAttr)
	logF.Close()

	if err != nil {
		return nil, fmt.Errorf("启动进程失败: %v", err)
	}

	return process, nil
}
