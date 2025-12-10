package tools

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// WaitTimeout 阻塞指定时间，并在信号不阻塞时提前退出
func WaitTimeout(done <-chan struct{}, d time.Duration) <-chan struct{} {
	out := make(chan struct{})
	timer := time.NewTimer(d)

	go func() {
		defer close(out)
		defer timer.Stop()

		select {
		case <-done:
			return
		case <-timer.C:
			return
		}
	}()

	return out
}

// IsValidForDial 检查网络地址是否合法
func IsValidForDial(addr string) bool {
	_, err := net.ResolveTCPAddr("tcp", addr)
	return err == nil
}

// EnsureFilePath 判断指定路径的文件是否存在，如果不存在就创建该路径
func EnsureFilePath(path string) (bool, error) {
	// 先判断文件是否存在
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat error: %w", err)
	}

	// 文件不存在，创建目录
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("mkdir error: %w", err)
	}

	return false, nil
}
