package singleinstance

import (
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAcquireTwice(t *testing.T) {
	g1, err := Acquire("test-instance")
	if err != nil {
		t.Fatalf("首次获取实例锁失败: %v", err)
	}
	defer g1.Release()

	if _, err := Acquire("test-instance"); err == nil {
		t.Fatal("第二次获取相同实例锁应失败")
	}
}

func TestAcquireAfterRelease(t *testing.T) {
	g1, err := Acquire("test-instance-release")
	if err != nil {
		t.Fatalf("首次获取实例锁失败: %v", err)
	}
	if err := g1.Release(); err != nil {
		t.Fatalf("释放实例锁失败: %v", err)
	}

	g2, err := Acquire("test-instance-release")
	if err != nil {
		t.Fatalf("释放后应可重新获取实例锁: %v", err)
	}
	defer g2.Release()
}

func TestAcquireCleansStaleLock(t *testing.T) {
	name := "stale-lock-instance"
	path := lockFilePath(name)
	_ = os.Remove(path)

	staleTs := time.Now().Add(-48 * time.Hour).Unix()
	content := "999999\n" + strconv.FormatInt(staleTs, 10) + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("写入陈旧锁失败: %v", err)
	}
	defer os.Remove(path)

	g, err := Acquire(name)
	if err != nil {
		t.Fatalf("应清理陈旧锁并成功获取: %v", err)
	}
	defer g.Release()
}
