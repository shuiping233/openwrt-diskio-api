//go:build linux
// +build linux

package metric

import (
	"testing"
)

func TestGetStatfsSuccess(t *testing.T) {
	// 测试根文件系统
	result, err := getStatfs("/")

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

func TestGetStatfsInvalidPath(t *testing.T) {
	testCases := map[string]string{
		"None exist path": "/nonexistent/path/that/does/not/exist",
		"Empty path":      "",
	}
	for testName, path := range testCases {
		t.Run(testName, func(t *testing.T) {
			_, err := getStatfs(path)
			if err == nil {
				t.Errorf("Expected error for invalid path %q , got nil", path)
			}
		})
	}
}


