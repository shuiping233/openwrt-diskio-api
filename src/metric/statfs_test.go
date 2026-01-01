//go:build linux
// +build linux

package metric

import (
	"testing"
)

func TestGetStatfs_Success(t *testing.T) {
	// 测试根文件系统
	result, err := GetStatfs("/")

	if err != nil {
		t.Fatalf("GetStatfs('/') failed: %v", err)
	}

	// 验证基本字段
	if result.Bsize == 0 {
		t.Error("Bsize should not be 0")
	}

	if result.Blocks == 0 {
		t.Error("Blocks should not be 0")
	}

	t.Logf("File system info - Block size: %d, Total blocks: %d, Free blocks: %d",
		result.Bsize, result.Blocks, result.Bfree)
}

func TestGetStatfs_InvalidPath(t *testing.T) {
	// 测试无效路径
	_, err := GetStatfs("/nonexistent/path/that/does/not/exist")

	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestGetStatfs_EmptyPath(t *testing.T) {
	// 测试空路径
	_, err := GetStatfs("")

	if err == nil {
		t.Error("Expected error for empty path, got nil")
	}
}
