package singleinstance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var ErrAlreadyRunning = errors.New("instance already running")

type Guard struct {
	path string
	file *os.File
}

func Acquire(name string) (*Guard, error) {
	path := lockFilePath(name)

	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = fmt.Fprintf(file, "%d\n%d\n", os.Getpid(), time.Now().Unix())
			return &Guard{path: path, file: file}, nil
		}
		if !os.IsExist(err) {
			return nil, fmt.Errorf("无法创建实例锁文件: %w", err)
		}

		active, checkErr := lockOwnedByActiveProcess(path)
		if checkErr != nil {
			return nil, checkErr
		}
		if active {
			return nil, ErrAlreadyRunning
		}

		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return nil, fmt.Errorf("清理陈旧锁文件失败: %w", removeErr)
		}
	}

	return nil, ErrAlreadyRunning
}

func (g *Guard) Release() error {
	if g == nil {
		return nil
	}
	if g.file != nil {
		_ = g.file.Close()
	}
	if g.path == "" {
		return nil
	}
	if err := os.Remove(g.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func lockOwnedByActiveProcess(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("读取锁文件失败: %w", err)
	}

	parts := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(parts) == 0 {
		return false, nil
	}

	pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || pid <= 0 {
		return false, nil
	}

	if len(parts) > 1 {
		if ts, parseErr := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); parseErr == nil {
			if time.Since(time.Unix(ts, 0)) > 24*time.Hour {
				return false, nil
			}
		}
	}

	return isProcessRunning(pid), nil
}

func lockFilePath(name string) string {
	safe := strings.ReplaceAll(name, string(os.PathSeparator), "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	if safe == "" {
		safe = "game-control"
	}
	return filepath.Join(os.TempDir(), safe+".lock")
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	if errors.Is(err, os.ErrProcessDone) {
		return false
	}

	if errno, ok := err.(syscall.Errno); ok {
		if errno == syscall.EPERM {
			return true
		}
		if errno == syscall.ESRCH {
			return false
		}
	}

	return false
}
